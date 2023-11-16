package main

import (
	//"log"
	"context"
	"strings"
	"sync"

	_ "api-rock-paper-scissors/docs"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

type Leaderboard struct {
	Name  string
	Score int
}

// SuccessResponse represents the successful response format
type SuccessResponse struct {
	SessionID int `json:"session_id"`
}

// ErrorResponse представляет собой структуру для ошибочных ответов API.
type ErrorResponse struct {
	Error string `json:"error"`
}

type PlayResponse struct {
	GameResult string `json:"game_result" example:"Win 1 player"`
}

func getDBConnection() (*pgx.Conn, error) {
	config, err := pgx.ParseConfig("postgres://postgres:123@localhost/api")
	if err != nil {
		return nil, err
	}
	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func closeDBConnection(conn *pgx.Conn) {
	conn.Close(context.Background())
}

func PostLeaderBoard(name string) error {

	conn, err := getDBConnection()
	if err != nil {
		panic(err)
	}
	defer closeDBConnection(conn)

	sqlupdate := `UPDATE leaderboard set score = score + $1 where name = $2`
	sqlinsert := `INSERT INTO leaderboard (score, name) VALUES ($1, $2)`
	println(name)
	_, err = conn.Exec(context.Background(), sqlupdate, 1, name)
	if err != nil {
		_, err = conn.Exec(context.Background(), sqlinsert, 1, name)
	}
	return err

}

func GetLeaderBoard() []Leaderboard {

	var leaderboard = []Leaderboard{}
	conn, err := getDBConnection()
	if err != nil {
		panic(err)
	}
	defer closeDBConnection(conn)

	sql := `SELECT name, score FROM leaderboard`

	var p Leaderboard

	rows, err := conn.Query(context.Background(), sql)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&p.Name, &p.Score)
		if err != nil {
			panic(err)
		}
		leaderboard = append(leaderboard, p)
	}
	return leaderboard

}

const MAX_ROUND_FILTER = 3

var ID = 1

var sessions []*Session

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

// @Description Создание новой игровой сессии
// @Tags Session
// @Accept json
// @Produce json
// @Summary Создание игровой сессии
// @Produce json
// @Success 200 {object} SuccessResponse "Successful response with session ID"
// @Router /create_session [post]
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

// @Description Присоединение к игровой сессии
// @Tags Session
// @Accept json
// @Produce json
// @Summary Присоединение к игровой сессии
// @Param session_id path int true "ID игровой сессии"
// @Param player_name path string true "Имя игрока"
// @Success 200 {object} SuccessResponse "Игрок успешно присоединился к сессии"
// @Failure 400 {object} ErrorResponse "Некорректный запрос или неверный ID сессии"
// @Router /join_session [post]
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

	session := sessions[ID]
	session.mu.Lock()
	defer session.mu.Unlock()

	var addPlayers Player
	addPlayers.Name = json.PlayerName
	addPlayers.Choice = ""

	session.Players = append(session.Players, addPlayers)
	c.JSON(200, gin.H{"message": "Player joined the session"})
}

// @Summary Получить текущий рейтинг игроков
// @Description Возвращает рейтинг игроков в текущих игровых сессиях.
// @Tags Session
// @Produce json
// @Success 200 {object} map[string]int "map[имя_игрока:количество_побед]"
// @Router /leaderboard [get]
func getLeaderboard(c *gin.Context) {
	c.JSON(200, GetLeaderBoard())
}

// @Summary Получить список текущих игровых сессий
// @Description Возвращает список текущих игровых сессий, включая информацию о каждой сессии.
// @Tags Session
// @Produce json
// @Success 200 {array} Session "Массив сессий"
// @Router /current_games [get]
func getCurrentGames(c *gin.Context) {
	c.JSON(200, gin.H{"current_games": sessions})
}

// @Summary Играть в рок-ножницы-бумага
// @Description Производит игровой ход в рамках сессии, включая выбор игрока и определение победителя в раунде.
// @Tags Session
// @Accept json
// @Produce json
// @Param session_id path int true "Идентификатор игровой сессии"
// @Param player_name path string true "Имя игрока"
// @Param choice path string true "Выбор игрока (rock, paper, scissors)"
// @Success 200 {object} PlayResponse "Результат игры"
// @Failure 400 {object} ErrorResponse "Некорректный запрос или неверный идентификатор сессии"
// @Router /play [post]
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

	session := sessions[ID]
	session.mu.Lock()
	defer session.mu.Unlock()

	for i, Player := range sessions[ID].Players { // Запись выбора игрока
		if strings.HasPrefix(Player.Name, json.PlayerName) {
			sessions[ID].Players[i].Choice = json.Choice
		}
	}

	var counter_answer int = 0 // Кол-во ответов

	for i := range sessions[ID].Players { // Подсчет количества ответов от игроков
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
			PostLeaderBoard(sessions[ID].Players[0].Name)
		} else {
			PostLeaderBoard(sessions[ID].Players[1].Name)
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

	// Маршрут для Swagger UI
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8000")
}
