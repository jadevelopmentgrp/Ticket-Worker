package messagequeue

import (
	"context"

	"github.com/TicketsBot/common/closerequest"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/cache"
	cmdcontext "github.com/jadevelopmentgrp/Tickets-Worker/bot/command/context"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/constants"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/dbclient"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/logic"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/metrics/statsd"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/redis"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/utils"
)

func ListenCloseRequestTimer() {
	ch := make(chan database.CloseRequest)
	go closerequest.Listen(redis.Client, ch)

	for request := range ch {
		statsd.Client.IncrementKey(statsd.AutoClose)

		request := request
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), constants.TimeoutCloseTicket)
			defer cancel()

			// get ticket
			ticket, err := dbclient.Client.Tickets.Get(ctx, request.TicketId, request.GuildId)
			if err != nil {
				sentry.Error(err)
				return
			}

			// get worker
			worker, err := buildContext(ctx, ticket, cache.Client)
			if err != nil {
				sentry.Error(err)
				return
			}

			// query already checks, but just to be sure
			if ticket.ChannelId == nil {
				return
			}

			// get premium status
			premiumTier, err := utils.PremiumClient.GetTierByGuildId(ctx, ticket.GuildId, true, worker.Token, worker.RateLimiter)
			if err != nil {
				sentry.Error(err)
				return
			}

			cc := cmdcontext.NewAutoCloseContext(ctx, worker, ticket.GuildId, *ticket.ChannelId, request.UserId, premiumTier)
			logic.CloseTicket(ctx, cc, request.Reason, true)
		}()
	}
}
