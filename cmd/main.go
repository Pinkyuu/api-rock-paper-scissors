package main

import (
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Player struct {
	Name   string
	Choice string
}

type Scores struct {
	player_1 int
	player_2 int
}

type Session struct {
	ID      int
	Players []Player
	Round   int
	Score   Scores
	mu      sync.Mutex
}

const MAX_ROUND_FILTER = 3

var ID = 1

var sessions []*Session
var leaderboard map[string]int

func findArrID(ID int) int { // Поиск игры по ID
	for i, session := range sessions {
		if session.ID == ID {
			return i
		}
	}
	return -1
}

func getResult(player_1 string, player_2 string) int {
	if player_1 == player_2 {
		return 0 // ничья
	} else if (player_1 == "rock" && player_2 == "scissors") ||
		(player_1 == "scissors" && player_2 == "paper") ||
		(player_1 == "paper" && player_2 == "rock") {
		return 1 // 1 игрок победил
	} else {
		return 2 // 2 игрок победил
	}
}

func createSession(c *gin.Context) {
	sessionID := ID
	ID++
	session := &Session{
		ID:      sessionID,
		Players: make([]Player, 0),
	}
	sessions = append(sessions, session)
	c.JSON(200, gin.H{"session_id": sessionID})

	println("Session created with ID: ", session.ID)

}

func joinSession(c *gin.Context) {
	var json struct {
		SessionID  int    `json:"session_id"`
		PlayerName string `json:"player_name"`
	}
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if json.SessionID <= 0 || json.SessionID > len(sessions) {
		c.JSON(400, gin.H{"error": "Invalid session ID"})
		return
	}

	var ID int = findArrID(json.SessionID) // Поиск номера игры в массиве (session) по ID

	if len(sessions[ID].Players) > 1 {
		c.JSON(400, gin.H{"error": "Max players"})
		return
	}

	session := sessions[json.SessionID-1]
	session.mu.Lock()
	defer session.mu.Unlock()

	var addPlayers Player
	addPlayers.Name = json.PlayerName
	addPlayers.Choice = ""

	session.Players = append(session.Players, addPlayers)
	c.JSON(200, gin.H{"message": "Player joined the session"})
}

func getLeaderboard(c *gin.Context) {
	c.JSON(200, leaderboard)
}

func getCurrentGames(c *gin.Context) {
	c.JSON(200, gin.H{"current_games": sessions})
}

func play(c *gin.Context) {
	var json struct {
		SessionID  int    `json:"session_id"`
		PlayerName string `json:"player_name"`
		Choice     string `json:"choice"`
	}
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if json.SessionID <= 0 || json.SessionID > len(sessions) {
		c.JSON(400, gin.H{"error": "Invalid session ID"})
		return
	}

	var ID int = findArrID(json.SessionID) // Поиск номера игры в массиве (session) по ID
	if ID < 0 {
		c.JSON(400, gin.H{"error": "Session ID not found"})
		return
	}

	session := sessions[json.SessionID-1]
	session.mu.Lock()
	defer session.mu.Unlock()

	for i, Player := range sessions[ID].Players { // Запись выбора игрока
		if strings.HasPrefix(Player.Name, json.PlayerName) {
			sessions[ID].Players[i].Choice = json.Choice
		}
	}

	var counter_answer int = 0 // Кол-во ответов

	for i, _ := range sessions[ID].Players { // Подсчет количества ответов от игроков
		if sessions[ID].Players[i].Choice != "" {
			counter_answer++
		}
	}

	if counter_answer != len(sessions[ID].Players) { // Сравнение кол-во ответов и игроков
		c.JSON(200, gin.H{"Game": "Wait the enemy "})
		return
	}

	switch getResult(sessions[ID].Players[0].Choice, sessions[ID].Players[1].Choice) {
	case 0:
		c.JSON(200, gin.H{"Game": "Draw"})
	case 1:
		{
			c.JSON(200, gin.H{"Game": "Win 1 player"})
			sessions[ID].Score.player_1++
		}
	case 2:
		{
			c.JSON(200, gin.H{"Game": "Win 2 player"})
			sessions[ID].Score.player_2++
		}
	}

	sessions[ID].Players[0].Choice = ""
	sessions[ID].Players[1].Choice = ""

	sessions[ID].Round++ // закончился раунд

	if sessions[ID].Round > MAX_ROUND_FILTER {
		if sessions[ID].Score.player_1 > sessions[ID].Score.player_2 {
			if leaderboard == nil {
				leaderboard = make(map[string]int)
			}
			leaderboard[sessions[ID].Players[0].Name]++
		} else {
			if leaderboard == nil {
				leaderboard = make(map[string]int)
			}
			leaderboard[sessions[ID].Players[1].Name]++
		}

		sessions = append(sessions[:json.SessionID-1], sessions[json.SessionID:]...)

	}
}

func main() {
	r := gin.Default()

	r.POST("/create_session", createSession)
	r.POST("/join_session", joinSession)
	r.GET("/leaderboard", getLeaderboard)
	r.GET("/current_games", getCurrentGames)
	r.POST("/play", play)

	r.Run(":8080")
}
