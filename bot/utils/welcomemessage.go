package utils

import (
	"context"
	"fmt"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/TicketsBot/worker"
	"github.com/TicketsBot/worker/bot/dbclient"
	"github.com/rxdn/gdl/objects/guild/emoji"
	"github.com/rxdn/gdl/objects/interaction/component"
	"github.com/rxdn/gdl/rest"
	"golang.org/x/sync/errgroup"
	"strconv"
	"strings"
	"sync"
	"time"
)

// returns msg id
func SendWelcomeMessage(worker *worker.Context, ticket database.Ticket, isPremium bool, subject string, panel *database.Panel) (uint64, error) {
	// Send welcome message
	var welcomeMessage string
	if panel == nil || panel.WelcomeMessage == nil {
		var err error
		welcomeMessage, err = dbclient.Client.WelcomeMessages.Get(ticket.GuildId)
		if err != nil {
			sentry.Error(err)
			welcomeMessage = "Thank you for contacting support.\nPlease describe your issue (and provide an invite to your server if applicable) and wait for a response."
		}
	} else {
		welcomeMessage = *panel.WelcomeMessage
	}

	// %average_response%
	if isPremium && strings.Contains(welcomeMessage, "%average_response%") {
		weeklyResponseTime, err := dbclient.Client.FirstResponseTime.GetAverage(ticket.GuildId, time.Hour*24*7)
		if err != nil {
			sentry.Error(err)
		} else {
			strings.Replace(welcomeMessage, "%average_response%", FormatTime(*weeklyResponseTime), -1)
		}
	}

	// variables
	welcomeMessage = doSubstitutions(welcomeMessage, worker, ticket)

	// Send welcome message
	embed := BuildEmbedRaw(worker, Green, subject, welcomeMessage, nil, isPremium)
	data := rest.CreateMessageData{
		Embed: embed,
		Components: []component.Component{
			component.BuildActionRow(
				component.BuildButton(component.Button{
					Label:    "Close",
					CustomId: "close",
					Style:    component.ButtonStyleDanger,
					Emoji:    emoji.Emoji{Name: "🔒"},
				}),
				component.BuildButton(component.Button{
					Label:    "Claim",
					CustomId: "claim",
					Style:    component.ButtonStyleSuccess,
					Emoji:    emoji.Emoji{Name: "🙋‍♂️"},
				})),
		},
	}

	// Should never happen
	if ticket.ChannelId == nil {
		return 0, fmt.Errorf("channel is nil")
	}

	msg, err := worker.CreateMessageComplex(*ticket.ChannelId, data)
	if err != nil {
		return 0, err
	}

	return msg.Id, nil
}

func doSubstitutions(welcomeMessage string, ctx *worker.Context, ticket database.Ticket) string {
	var lock sync.Mutex

	// do DB lookups in parallel
	group, _ := errgroup.WithContext(context.Background())
	for placeholder, f := range substitutions {
		placeholder := placeholder
		f := f

		formatted := fmt.Sprintf("%%%s%%", placeholder)

		if strings.Contains(welcomeMessage, formatted) {
			group.Go(func() error {
				replacement := f(ctx, ticket)

				lock.Lock()
				welcomeMessage = strings.Replace(welcomeMessage, formatted, replacement, -1)
				lock.Unlock()

				return nil
			})
		}
	}

	if err := group.Wait(); err != nil {
		sentry.Error(err)
	}

	return welcomeMessage
}

var substitutions = map[string]func(ctx *worker.Context, ticket database.Ticket) string{
	"user": func(ctx *worker.Context, ticket database.Ticket) string {
		return fmt.Sprintf("<@%d>", ticket.UserId)
	},
	"ticket_id": func(ctx *worker.Context, ticket database.Ticket) string {
		return strconv.Itoa(ticket.Id)
	},
	"channel": func(ctx *worker.Context, ticket database.Ticket) string {
		return fmt.Sprintf("<#%d>", ticket.ChannelId)
	},
	"username": func(ctx *worker.Context, ticket database.Ticket) string {
		user, _ := ctx.GetUser(ticket.UserId)
		return user.Username
	},
	"server": func(ctx *worker.Context, ticket database.Ticket) string {
		guild, _ := ctx.GetGuild(ticket.GuildId)
		return guild.Name
	},
	"open_tickets": func(ctx *worker.Context, ticket database.Ticket) string {
		open, _ := dbclient.Client.Tickets.GetGuildOpenTickets(ticket.GuildId)
		return strconv.Itoa(len(open))
	},
	"total_tickets": func(ctx *worker.Context, ticket database.Ticket) string {
		total, _ := dbclient.Client.Tickets.GetTotalTicketCount(ticket.GuildId)
		return strconv.Itoa(total)
	},
	"user_open_tickets": func(ctx *worker.Context, ticket database.Ticket) string {
		tickets, _ := dbclient.Client.Tickets.GetOpenByUser(ticket.GuildId, ticket.UserId)
		return strconv.Itoa(len(tickets))
	},
	"ticket_limit": func(ctx *worker.Context, ticket database.Ticket) string {
		limit, _ := dbclient.Client.TicketLimit.Get(ticket.GuildId)
		return strconv.Itoa(int(limit))
	},
	"rating_count": func(ctx *worker.Context, ticket database.Ticket) string {
		ratingCount, _ := dbclient.Client.ServiceRatings.GetCount(ticket.GuildId)
		return strconv.Itoa(ratingCount)
	},
	"average_rating": func(ctx *worker.Context, ticket database.Ticket) string {
		average, _ := dbclient.Client.ServiceRatings.GetAverage(ticket.GuildId)
		return fmt.Sprintf("%.1f", average)
	},
}
