package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type MatchHandler struct {
	db *sql.DB
}

func NewMatchHandler(db *sql.DB) *MatchHandler {
	return &MatchHandler{db: db}
}
func (h *MatchHandler) GetMatches(w http.ResponseWriter, r *http.Request) {
	leagueID := r.URL.Query().Get("league_id")
	season := r.URL.Query().Get("season")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := `
		SELECT 
			m.id,
			TO_CHAR(m.date, 'YYYY-MM-DD') AS match_date, -- Только дата (например, 2023-11-11)
			TO_CHAR(m.date, 'HH24:MI') AS match_time, -- Только время (например, 17:30)
			home_team.id AS home_team_id,
			home_team.fullname AS home_team,
			away_team.id AS away_team_id,
			away_team.fullname AS away_team,
			m.home_score,
			m.away_score
		FROM matches m
		JOIN teams home_team ON m.home_team_id = home_team.id
		JOIN teams away_team ON m.away_team_id = away_team.id
		WHERE m.league_id = $1 AND m.season = $2
		ORDER BY m.date DESC -- ← Сохраняем сортировку по дате
		LIMIT $3 OFFSET $4
    `

	rows, err := h.db.Query(query, leagueID, season, limit, offset)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var leagueName string
	err = h.db.QueryRow("SELECT fullname FROM leagues WHERE id = $1", leagueID).Scan(&leagueName)
	if err != nil {
		http.Error(w, "League not found", http.StatusNotFound)
		return
	}
	leagueIcon := fmt.Sprintf("https://media.api-sports.io/football/leagues/%s.png", leagueID)

	var matches []map[string]interface{}
	for rows.Next() {
		var (
			id           int
			matchDate    string
			matchTime    string
			homeTeamID   int
			homeTeamName string
			awayTeamID   int
			awayTeamName string
			homeScore    int
			awayScore    int
		)
		if err := rows.Scan(
			&id, &matchDate, &matchTime,
			&homeTeamID, &homeTeamName,
			&awayTeamID, &awayTeamName,
			&homeScore, &awayScore,
		); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		matches = append(matches, map[string]interface{}{
			"id":         id,
			"date":       matchDate,
			"time":       matchTime,
			"home_team":  homeTeamName,
			"away_team":  awayTeamName,
			"home_score": homeScore,
			"away_score": awayScore,
			"home_icon":  fmt.Sprintf("https://media.api-sports.io/football/teams/%d.png", homeTeamID),
			"away_icon":  fmt.Sprintf("https://media.api-sports.io/football/teams/%d.png", awayTeamID),
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"league": map[string]interface{}{
			"id":   leagueID,
			"name": leagueName,
			"icon": leagueIcon,
		},
		"matches": matches,
	})
}
func (h *MatchHandler) GetTopTeams(w http.ResponseWriter, r *http.Request) {
	season := r.URL.Query().Get("season")

	query := `
        WITH team_stats AS (
            SELECT
                t.id AS team_id,
                t.fullname AS team_name,
                COUNT(m.id) AS matches_played,
                COALESCE(SUM(
                    CASE 
                        WHEN (m.home_team_id = t.id AND m.home_score > m.away_score) OR 
                             (m.away_team_id = t.id AND m.away_score > m.home_score) 
                        THEN 3
                        WHEN m.home_score = m.away_score THEN 1
                        ELSE 0
                    END
                ), 0) AS total_points,
                COALESCE(SUM(
                    CASE 
                        WHEN m.home_team_id = t.id THEN m.home_score
                        WHEN m.away_team_id = t.id THEN m.away_score
                        ELSE 0
                    END
                ), 0) AS goals_for,
                COALESCE(SUM(
                    CASE 
                        WHEN m.home_team_id = t.id THEN m.away_score
                        WHEN m.away_team_id = t.id THEN m.home_score
                        ELSE 0
                    END
                ), 0) AS goals_against
            FROM teams t
            LEFT JOIN matches m ON (t.id = m.home_team_id OR t.id = m.away_team_id) AND m.season = $1
            GROUP BY t.id
        )
        SELECT 
            team_id,
            team_name,
            ROUND(total_points / NULLIF(matches_played, 0)::numeric, 2) AS avg_points_per_game,
            ROUND(goals_for / NULLIF(matches_played, 0)::numeric, 2) AS avg_goals_for,
            ROUND(goals_against / NULLIF(matches_played, 0)::numeric, 2) AS avg_goals_against
        FROM team_stats
        ORDER BY 
            avg_points_per_game DESC,
            avg_goals_for DESC,
            avg_goals_against ASC
        LIMIT 10
    `

	rows, err := h.db.Query(query, season)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	teams := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			teamID          int
			teamName        string
			avgPoints       sql.NullFloat64
			avgGoalsFor     sql.NullFloat64
			avgGoalsAgainst sql.NullFloat64
		)
		if err := rows.Scan(&teamID, &teamName, &avgPoints, &avgGoalsFor, &avgGoalsAgainst); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		teams = append(teams, map[string]interface{}{
			"id":                teamID,
			"name":              teamName,
			"avg_points":        avgPoints.Float64,
			"avg_goals_for":     avgGoalsFor.Float64,
			"avg_goals_against": avgGoalsAgainst.Float64,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"season": season,
		"teams":  teams,
	})
}
func (h *MatchHandler) GetTopPlayers(w http.ResponseWriter, r *http.Request) {
	season := r.URL.Query().Get("season")

	query := `
        SELECT 
            t.id AS team_id,
            p.fullname AS player_name,
            SUM(l.goals + l.assists) AS total_points,
            SUM(l.goals) AS total_goals,
            SUM(l.assists) AS total_assists,
            ROUND(SUM(l.minutes)::numeric / NULLIF(SUM(l.goals + l.assists), 0), 2) AS avg_points_time,
            ROUND(SUM(l.minutes)::numeric / NULLIF(SUM(l.goals), 0), 2) AS avg_goal_time
        FROM players p
        JOIN lineups l ON p.id = l.player_id
        JOIN matches m ON l.match_id = m.id
        JOIN teams t ON l.team_id = t.id
        WHERE m.season = $1
        GROUP BY p.id, t.id
        HAVING SUM(l.goals) + SUM(l.assists) > 0
        ORDER BY 
            total_points DESC,
            total_goals DESC,
            total_assists DESC,
            avg_points_time ASC,
			avg_goal_time ASC
        LIMIT 10
    `

	rows, err := h.db.Query(query, season)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	players := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			teamID        int
			playerName    string
			totalPoints   int
			goals         int
			assists       int
			avgPointsTime sql.NullFloat64
			avgGoalTime   sql.NullFloat64
		)
		if err := rows.Scan(
			&teamID, &playerName,
			&totalPoints, &goals, &assists,
			&avgPointsTime, &avgGoalTime,
		); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		players = append(players, map[string]interface{}{
			"team_id":       teamID,
			"name":          playerName,
			"total_points":  totalPoints,
			"total_goals":   goals,
			"total_assists": assists,
			"avg_points":    avgPointsTime.Float64,
			"avg_goal":      avgGoalTime.Float64,
		})
	}

	json.NewEncoder(w).Encode(players)
}
func (h *MatchHandler) GetLeagues(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, fullname, country FROM leagues")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var leagues []map[string]interface{}
	for rows.Next() {
		var id int
		var name, country string
		if err := rows.Scan(&id, &name, &country); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		iconURL := fmt.Sprintf("https://media.api-sports.io/football/leagues/%d.png", id)

		leagues = append(leagues, map[string]interface{}{
			"id":      id,
			"name":    name,
			"country": country,
			"icon":    iconURL,
		})
	}

	json.NewEncoder(w).Encode(leagues)
}

func (h *MatchHandler) GetSeasons(w http.ResponseWriter, r *http.Request) {
	query := `
        SELECT DISTINCT season 
        FROM league_seasons 
        ORDER BY season DESC
    `

	rows, err := h.db.Query(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	type Season struct {
		Value string `json:"value"` // Исходный сезон (2023)
		Label string `json:"label"` // Отображаемый сезон (2023/24)
	}
	var seasons []Season
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		seasons = append(seasons, Season{
			Value: s,
			Label: formatSeason(s),
		})
	}

	json.NewEncoder(w).Encode(seasons)
}

/*func (h *MatchHandler) GetTopTeams(w http.ResponseWriter, r *http.Request) {
	query := `
        WITH team_matches AS (
            SELECT
                t.id AS team_id,
                t.fullname AS team_name,
                m.date,
                m.home_team_id,
                m.away_team_id,
                m.home_score,
                m.away_score,
                ROW_NUMBER() OVER (PARTITION BY t.id ORDER BY m.date DESC) AS match_num
            FROM teams t
            LEFT JOIN matches m ON t.id = m.home_team_id OR t.id = m.away_team_id
        ),
        recent_matches AS (
            SELECT
                team_id,
                team_name,
                date,
                home_team_id,
                away_team_id,
                home_score,
                away_score
            FROM team_matches
            WHERE match_num <= 5
        ),
        team_stats AS (
            SELECT
                team_id,
                team_name,
                COUNT(*) AS total_matches,
                SUM(
                    CASE
                        WHEN (home_team_id = team_id AND home_score > away_score) OR
                             (away_team_id = team_id AND away_score > home_score)
                        THEN 3
                        WHEN home_score = away_score THEN 1
                        ELSE 0
                    END
                ) AS points,
                ARRAY_AGG(
                    CASE
                        WHEN (home_team_id = team_id AND home_score > away_score) OR
                             (away_team_id = team_id AND away_score > home_score)
                        THEN 'W'
                        WHEN home_score = away_score THEN 'D'
                        ELSE 'L'
                    END
                    ORDER BY date DESC
                ) AS form
            FROM recent_matches
            GROUP BY team_id, team_name
        )
        SELECT
            team_id,
            team_name,
            points,
            form
        FROM team_stats
        ORDER BY points DESC
        LIMIT 10
    `
	rows, err := h.db.Query(query)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	teams := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			teamID   int
			teamName string
			points   int
			form     []string
		)
		if err := rows.Scan(&teamID, &teamName, &points, pq.Array(&form)); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		teams = append(teams, map[string]interface{}{
			"id":     teamID,
			"name":   teamName,
			"points": points,
			"form":   form,
		})
	}

	json.NewEncoder(w).Encode(teams)
}
*/
