package listeners

import (
	"context"
	"time"

	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/worker"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/dbclient"
	"github.com/jadevelopmentgrp/Tickets-Worker/bot/errorcontext"
	"github.com/rxdn/gdl/gateway/payloads/events"
	"golang.org/x/sync/errgroup"
)

func OnRoleDelete(worker *worker.Context, e events.GuildRoleDelete) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3) // TODO: Propagate context
	defer cancel()

	errorCtx := errorcontext.WorkerErrorContext{Guild: e.GuildId}

	group, _ := errgroup.WithContext(context.Background())

	group.Go(func() error {
		return dbclient.Client.RolePermissions.RemoveSupport(ctx, e.GuildId, e.RoleId)
	})

	group.Go(func() error {
		return dbclient.Client.SupportTeamRoles.DeleteAllRole(ctx, e.RoleId)
	})

	group.Go(func() error {
		return dbclient.Client.PanelRoleMentions.DeleteAllRole(ctx, e.RoleId)
	})

	if err := group.Wait(); err != nil {
		sentry.ErrorWithContext(err, errorCtx)
	}
}
