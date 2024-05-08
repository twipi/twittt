package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/twipi/pubsub"
	"github.com/twipi/twipi/proto/out/twicmdproto"
	"github.com/twipi/twipi/proto/out/twismsproto"
	"github.com/twipi/twipi/twicmd"
	"github.com/twipi/twipi/twisms"
	"github.com/twipi/twittt/game"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/prototext"
)

//go:embed service.txtpb
var servicePrototext []byte

var service = (func() *twicmdproto.Service {
	service := new(twicmdproto.Service)
	if err := prototext.Unmarshal(servicePrototext, service); err != nil {
		panic(fmt.Sprintf("failed to unmarshal service proto: %v", err))
	}
	return service
})()

const gameExpiry = 24 * time.Hour

type runningGame struct {
	*game.Game
	AI        *game.AI
	StartedAt time.Time
}

// Service is the main running Tic-tac-toe Twicmd service.
type Service struct {
	sendCh  chan *twismsproto.Message
	sendSub pubsub.Subscriber[*twismsproto.Message]
	games   *xsync.MapOf[string, *runningGame]
	logger  *slog.Logger
}

var (
	_ twicmd.Service           = (*Service)(nil)
	_ twisms.MessageSubscriber = (*Service)(nil)
)

func NewService(logger *slog.Logger) *Service {
	return &Service{
		sendCh: make(chan *twismsproto.Message),
		games:  xsync.NewMapOf[string, *runningGame](),
		logger: logger,
	}
}

// Name implements [twicmd.Service].
func (s *Service) Name() string {
	return service.Name
}

// Service implements [twicmd.Service].
func (s *Service) Service(ctx context.Context) (*twicmdproto.Service, error) {
	return service, nil
}

// Execute implements [twicmd.Service].
func (s *Service) Execute(ctx context.Context, req *twicmdproto.ExecuteRequest) (*twicmdproto.ExecuteResponse, error) {
	switch req.Command.Command {
	case "start":
		s.logger.Debug(
			"starting new game",
			"phone_number", req.Message.From)

		gm := game.NewGame()
		ai := game.NewAI(gm, game.Player2)

		_, overridden := s.games.LoadAndStore(req.Message.From, &runningGame{
			Game:      gm,
			AI:        ai,
			StartedAt: time.Now(),
		})

		if overridden {
			s.sendCh <- twisms.NewReplyingMessage(req.Message, twisms.NewTextBody(
				"An existing game was overridden. A new game has started. It is now your turn.",
			))
		} else {
			s.sendCh <- twisms.NewReplyingMessage(req.Message, twisms.NewTextBody(
				"A new game has started. It is now your turn.",
			))
		}

		s.sendCh <- twisms.NewReplyingMessage(req.Message, drawBoardMessage("", gm.Board))
		return nil, nil

	case "place":
		args := twicmd.MapArguments(req.Command.Arguments)
		s.logger.Debug(
			"placing piece",
			"phone_number", req.Message.From,
			"position", args["position"])

		gm, ok := s.games.Load(req.Message.From)
		if !ok {
			return twicmd.StatusResponse("No game found. Please start a new game."), nil
		}

		npos, err1 := strconv.Atoi(args["position"])
		bpos, err2 := game.BoardPositionFromIndex(npos - 1)
		if err1 != nil || err2 != nil {
			return twicmd.StatusResponse("Invalid position. Please provide a number between 1 and 9."), nil
		}

		msgs := []string{
			"You just placed:",
			"In return, the AI placed:",
		}

		for i, move := range []func() bool{
			func() bool { return gm.MakeMove(bpos) },
			func() bool { return gm.AI.MakeMove() },
		} {
			ok := move()
			s.sendCh <- twisms.NewReplyingMessage(req.Message, drawBoardMessage(msgs[i], gm.Board))
			if !ok {
				if winner, ended := gm.GameState(); ended {
					var msg string
					if winner != game.NoPlayer {
						msg = fmt.Sprintf("The game is over. %s wins!", playerUnicode[winner])
					} else {
						msg = "The game is over. It's a draw!"
					}
					return twicmd.TextResponse(msg), nil
				}
				return twicmd.StatusResponse("Invalid move. Please try again."), nil
			}
		}

		return nil, nil

	default:
		return nil, fmt.Errorf("unknown command: %q", req.Command.Command)
	}
}

var playerUnicode = map[game.Player]string{
	game.Player1:  "❌",
	game.Player2:  "⚫",
	game.NoPlayer: "⬜",
}

func drawBoardMessage(prefix string, board game.Board) *twismsproto.MessageBody {
	var s strings.Builder
	if prefix != "" {
		s.WriteString(prefix)
		s.WriteString("\n\n")
	}
	for r := range 3 {
		for c := range 3 {
			s.WriteString(playerUnicode[board[r][c]])
		}
		s.WriteString("\n")
	}
	s.WriteString("❌ is your piece.\n")
	s.WriteString("⚫ is the AI's piece.")
	return twisms.NewTextBody(s.String())
}

func (s *Service) Start(ctx context.Context) error {
	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() error {
		return s.sendSub.Listen(ctx, s.sendCh)
	})

	errg.Go(func() error {
		ticker := time.NewTicker(4 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case now := <-ticker.C:
				// clean up
				s.games.Range(func(key string, value *runningGame) bool {
					if value.StartedAt.Add(gameExpiry).Before(now) {
						s.logger.Debug(
							"game expired, deleting",
							"phone_number", key,
							"started_at", value.StartedAt)
						s.games.Delete(key)
					}
					return true
				})
			}
		}
	})

	return errg.Wait()
}

// SubscribeMessages implements [twisms.MessageSubscriber].
func (s *Service) SubscribeMessages(ch chan<- *twismsproto.Message, filters *twismsproto.MessageFilters) {
	s.sendSub.Subscribe(ch, func(msg *twismsproto.Message) bool {
		return twisms.FilterMessage(filters, msg)
	})
}

// UnsubscribeMessages implements [twisms.MessageSubscriber].
func (s *Service) UnsubscribeMessages(ch chan<- *twismsproto.Message) {
	s.sendSub.Unsubscribe(ch)
}
