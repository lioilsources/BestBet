package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/hockey"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/nhl"
	"github.com/lioilsources/BestBet/internal/store"
)

// NHL walks one season day by day through the public score feed (one request
// per day) and, for finished games, the boxscore goalies and skaters. Days older than two
// days are cached forever, recent ones for an hour.
func NHL(ctx context.Context, e Env, hc *httpcache.Client, l config.League, row dbq.League, season int, withGoalies bool) (Stats, error) {
	var st Stats
	teams, err := e.Store.LoadTeams(ctx, row.ID, "nhl", e.aliases("nhl", l.Slug))
	if err != nil {
		return st, err
	}
	players, err := e.Store.LoadPlayers(ctx, "nhl")
	if err != nil {
		return st, err
	}
	now := e.Now()
	from := time.Date(season, time.September, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(season+1, time.June, 30, 0, 0, 0, 0, time.UTC)
	if lim := now.AddDate(0, 0, 14); to.After(lim) {
		to = lim
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		ttl := time.Duration(0)
		if now.Sub(d) < 48*time.Hour {
			ttl = time.Hour
		}
		u := fmt.Sprintf("%s/score/%s", nhl.Base, d.Format("2006-01-02"))
		b, _, err := hc.Get(ctx, httpcache.Request{URL: u, Key: u, TTL: ttl})
		if err != nil {
			return st, fmt.Errorf("nhl %s: %w", d.Format("2006-01-02"), err)
		}
		games, err := nhl.ParseScore(b)
		if err != nil {
			return st, err
		}
		for _, g := range games {
			phase, ok := g.Phase()
			if !ok || g.Season != season {
				continue
			}
			st.Seen++
			home, err := teams.Resolve(ctx, g.Home.Name, 0)
			if err != nil {
				return st, err
			}
			away, err := teams.Resolve(ctx, g.Away.Name, 0)
			if err != nil {
				return st, err
			}
			in := storeInput(row.ID, season, g, phase, home, away)
			if in.Status == "finished" {
				r, err := g.Result()
				if err != nil {
					st.Failed++
					slog.Warn("nhl result", "game", g.ID, "err", err)
				} else {
					in.Result = &r
					in.OTType = hockey.OTType(r, phase, l.OT)
				}
			}
			gameID, method, err := e.Store.UpsertNHLGame(ctx, g.ID, in)
			if err != nil {
				return st, err
			}
			if method == "new" {
				st.New++
			} else {
				st.Linked++
			}
			if err := e.Store.Link(ctx, "nhl", strconv.FormatInt(g.ID, 10), row.ID, gameID, method, g.Home.Abbrev+" - "+g.Away.Abbrev); err != nil {
				return st, err
			}
			if !withGoalies || in.Status != "finished" {
				continue
			}
			n, k, err := nhlBoxscore(ctx, e, hc, players, gameID, home, away, g.ID)
			if errors.Is(err, httpcache.ErrBudget) {
				return st, err
			}
			if err != nil {
				st.Failed++
				slog.Warn("nhl boxscore", "game", g.ID, "err", err)
				continue
			}
			st.Goalies += n
			st.Skaters += k
		}
	}
	if len(teams.Created) > 0 {
		slog.Info("teams created", "league", l.Slug, "names", teams.Created)
	}
	if _, err := e.Store.DeriveSeries(ctx, row, int32(season)); err != nil {
		return st, err
	}
	slog.Info("nhl", "league", l.Slug, "season", season, "stats", st.String(), "players_new", players.New, "requests", hc.Used())
	return st, nil
}

// nhlBoxscore stores the goalies and skaters of one finished game from its
// boxscore (cached forever once the game is final).
func nhlBoxscore(ctx context.Context, e Env, hc *httpcache.Client, pl *store.Players, gameID int64, home, away int32, nhlID int64) (goalies, skaters int, err error) {
	u := fmt.Sprintf("%s/gamecenter/%d/boxscore", nhl.Base, nhlID)
	b, _, err := hc.Get(ctx, httpcache.Request{URL: u, Key: u})
	if err != nil {
		return 0, 0, err
	}
	gs, err := nhl.ParseBoxscore(b)
	if err != nil {
		return 0, 0, err
	}
	for _, g := range gs {
		team := home
		if g.Side == "away" {
			team = away
		}
		sa, sv, ga, toi := int32(g.ShotsAgainst), int32(g.Saves), int32(g.GoalsAgainst), int32(g.TOISeconds)
		var pct *float64
		if g.ShotsAgainst > 0 {
			p := float64(g.Saves) / float64(g.ShotsAgainst)
			pct = &p
		}
		if err := e.Store.Q.UpsertGoalie(ctx, dbq.UpsertGoalieParams{
			GameID: gameID, TeamID: team, Side: g.Side, PlayerName: g.Name, Starter: g.Starter,
			ShotsAgainst: &sa, Saves: &sv, GoalsAgainst: &ga, SavePct: pct, ToiSeconds: &toi,
			ConfirmStatus: "CONFIRMED", Source: "nhl",
		}); err != nil {
			return 0, 0, fmt.Errorf("goalie: %w", err)
		}
	}
	ss, err := nhl.ParseSkaters(b)
	if err != nil {
		return len(gs), 0, err
	}
	rows := make([]dbq.UpsertPlayerMatchStatsBatchParams, 0, len(ss))
	for _, x := range ss {
		id, err := pl.Resolve(ctx, strconv.Itoa(x.PlayerID), x.Name, x.Position)
		if err != nil {
			return len(gs), 0, err
		}
		team := home
		if x.Side == "away" {
			team = away
		}
		rows = append(rows, dbq.UpsertPlayerMatchStatsBatchParams{
			PlayerID: id, GameID: gameID, TeamID: team, Side: x.Side, Jersey: i32(x.Jersey), Position: &x.Position,
			ToiSeconds: i32(x.TOISeconds), Goals: i32(x.Goals), Assists: i32(x.Assists), PlusMinus: i32(x.PlusMinus),
			Shots: i32(x.Shots), Pim: i32(x.PIM), Source: "nhl",
		})
	}
	if err := e.Store.UpsertPlayerLines(ctx, rows); err != nil {
		return len(gs), 0, err
	}
	return len(gs), len(rows), nil
}

func i32(v int) *int32 {
	x := int32(v)
	return &x
}

func storeInput(leagueID int32, season int, g nhl.Game, phase string, home, away int32) store.GameInput {
	stage := "Regular Season"
	if phase != hockey.PhaseRegular {
		stage = "Playoffs"
	}
	return store.GameInput{LeagueID: leagueID, Season: int32(season), StartAt: g.StartAt, Status: g.Status(),
		HomeTeamID: home, AwayTeamID: away, Phase: phase, StageRaw: stage}
}
