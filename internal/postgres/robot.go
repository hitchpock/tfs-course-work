package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/hitchpock/tfs-course-work/internal/robot"
)

var _ robot.Storage = &RobotStorage{}

type RobotStorage struct {
	statementStorage

	createStmt                  *sql.Stmt
	findByIDStmt                *sql.Stmt
	findActivatedByUserIDStmt   *sql.Stmt
	findActivatedByTicker       *sql.Stmt
	findActivatedStmt           *sql.Stmt
	findActivatedByTickerUserID *sql.Stmt
	activateRobotStmt           *sql.Stmt
	deactivateRobotStmt         *sql.Stmt
	findToTradingStmt           *sql.Stmt
	tradeStmt                   *sql.Stmt
	softDeleteStmt              *sql.Stmt
}

// NewRobotStorage возвращает указатель на хранилище робтов.
func NewRobotStorage(db *DB) (*RobotStorage, error) {
	s := &RobotStorage{statementStorage: newStatementStorage(db)}

	stmts := []stmt{
		{Query: createRobotQuery, Dst: &s.createStmt},
		{Query: findByIDQuery, Dst: &s.findByIDStmt},
		{Query: softDeleteQuery, Dst: &s.softDeleteStmt},
		{Query: findActivatedByUserIDQuery, Dst: &s.findActivatedByUserIDStmt},
		{Query: findActivatedByTickerQuery, Dst: &s.findActivatedByTicker},
		{Query: findActivatedByTickerUserIDQuery, Dst: &s.findActivatedByTickerUserID},
		{Query: activateRobotQuery, Dst: &s.activateRobotStmt},
		{Query: deactivateRobotQuery, Dst: &s.deactivateRobotStmt},
		{Query: findActivatedQuery, Dst: &s.findActivatedStmt},
		{Query: tradeQuery, Dst: &s.tradeStmt},
		{Query: findToTradingQuery, Dst: &s.findToTradingStmt},
	}

	if err := s.initStatements(stmts); err != nil {
		return nil, fmt.Errorf("can't init statements: %s", err)
	}

	return s, nil
}

const robotFieldsInsert = `owner_user_id, parent_robot_id, is_favourite, is_active, ticker, buy_price, ` + //nolint:misspell
	`sell_price, plan_start, plan_end, plan_yield, fact_yield, deals_count, activated_at, deactivated_at, ` +
	`created_at, deleted_at, is_buying`

const robotFieldsSelect = `robot_id, ` + robotFieldsInsert

// scnaRobot сканирует робота из курсора базы данных.
func scanRobot(scanner sqlScanner, r *robot.Robot) error {
	return scanner.Scan(&r.RobotID, &r.OwnerUserID, &r.ParentRobotID, &r.IsFavourite, &r.IsActive, &r.Ticker,
		&r.BuyPrice, &r.SellPrice, &r.PlanStart, &r.PlanEnd, &r.PlanYield, &r.FactYield, &r.DealsCount,
		&r.ActivatedAt, &r.DeactivatedAt, &r.CreatedAt, &r.DeletedAt, &r.IsBuying)
}

// scanRobots возвращает список роботов из базы данных.
func scanRobots(rows *sql.Rows) ([]robot.Robot, error) {
	var robots []robot.Robot

	var err error

	for rows.Next() {
		var r robot.Robot
		if err = scanRobot(rows, &r); err != nil {
			return nil, fmt.Errorf("can't scan robot: %s", err)
		}

		robots = append(robots, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows return error: %s", err)
	}

	return robots, nil
}

const createRobotQuery = `INSERT INTO robots(` + robotFieldsInsert + `) VALUES ($1, $2, $3, $4, $5, $6, ` +
	`$7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

// Create дабавляет робота в хранилище.
func (s *RobotStorage) Create(r *robot.Robot) error {
	r.IsBuying = true
	r.CreatedAt.Valid = true
	r.CreatedAt.Time = time.Now()

	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("can't start a transaction: %s", err)
	}

	_, err = tx.Stmt(s.createStmt).Exec(r.OwnerUserID, r.ParentRobotID, r.IsFavourite, r.IsActive, r.Ticker, r.BuyPrice,
		r.SellPrice, r.PlanStart, r.PlanEnd, r.PlanYield, r.FactYield, r.DealsCount, r.ActivatedAt, r.DeactivatedAt,
		r.CreatedAt, r.DeletedAt, r.IsBuying)

	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("can't create robot: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in robotStorage: %s", err)
	}

	return nil
}

const findByIDQuery = `SELECT ` + robotFieldsSelect + ` FROM robots WHERE robot_id = $1 AND deleted_at IS NULL`

// FindByID находит робота по его ID.
func (s *RobotStorage) FindByID(id int) (*robot.Robot, error) {
	var r robot.Robot

	row := s.findByIDStmt.QueryRow(id)
	if err := scanRobot(row, &r); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: can't scan robot: %s", robot.ErrNotFound, err)
		}

		return nil, fmt.Errorf("can't scan robot: %s", err)
	}

	return &r, nil
}

const softDeleteQuery = `UPDATE robots SET deleted_at = $1 WHERE robot_id = $2`

// SoftDelete реализует 'soft delete', проставляя дату удаления, но не удаляя запись.
func (s *RobotStorage) SoftDelete(id int) error {
	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("can't start a transaction: %s", err)
	}

	if _, err = tx.Stmt(s.softDeleteStmt).Exec(time.Now(), id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("can't 'soft delete' robot: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in robotStorage: %s", err)
	}

	return nil
}

const findActivatedByUserIDQuery = `SELECT ` + robotFieldsSelect + ` FROM robots WHERE owner_user_id = $1 AND deleted_at IS NULL ORDER BY robot_id`

// FindActivatedByUserID находит активных роботов по пользовательскому ID.
func (s *RobotStorage) FindActivatedByUserID(id int) ([]robot.Robot, error) {
	rows, err := s.findActivatedByUserIDStmt.Query(id)
	if err != nil {
		return nil, fmt.Errorf("can't exec query: %s", err)
	}
	defer rows.Close()

	robots, err := scanRobots(rows)
	if err != nil {
		return nil, fmt.Errorf("can't scan robots: %s", err)
	}

	return robots, nil
}

const findActivatedByTickerQuery = `SELECT ` + robotFieldsSelect + ` FROM robots WHERE ticker = $1 AND deleted_at IS NULL ORDER BY robot_id`

// FindActivatedByTicker находит активных роботов по тикеру.
func (s *RobotStorage) FindActivatedByTicker(ticker string) ([]robot.Robot, error) {
	rows, err := s.findActivatedByTicker.Query(ticker)
	if err != nil {
		return nil, fmt.Errorf("can't exec query: %s", err)
	}
	defer rows.Close()

	robots, err := scanRobots(rows)
	if err != nil {
		return nil, fmt.Errorf("can't scan robots: %s", err)
	}

	return robots, nil
}

const findActivatedQuery = `SELECT ` + robotFieldsSelect + ` FROM robots WHERE deleted_at IS NULL ORDER BY robot_id`

// FindActivated возвращает список всех роботов.
func (s *RobotStorage) FindActivated() ([]robot.Robot, error) {
	rows, err := s.findActivatedStmt.Query()
	if err != nil {
		return nil, fmt.Errorf("can't exec query: %s", err)
	}
	defer rows.Close()

	robots, err := scanRobots(rows)
	if err != nil {
		return nil, fmt.Errorf("can't scan robots: %s", err)
	}

	return robots, nil
}

const findActivatedByTickerUserIDQuery = `SELECT ` + robotFieldsSelect + ` FROM robots WHERE ticker = $1 AND owner_user_id = $2 AND deleted_at IS NULL ORDER BY robot_id`

// FindActivatedByTickerUserID возвращает список активных роблтов по тикеру и пользовательскому id.
func (s *RobotStorage) FindActivatedByTickerUserID(ticker string, userID int) ([]robot.Robot, error) {
	rows, err := s.findActivatedByTickerUserID.Query(ticker, userID)
	if err != nil {
		return nil, fmt.Errorf("can't exec query: %s", err)
	}
	defer rows.Close()

	robots, err := scanRobots(rows)
	if err != nil {
		return nil, fmt.Errorf("can't scan robots: %s", err)
	}

	return robots, nil
}

// Filter выбирает какой запрос необходимо обработать.
func (s *RobotStorage) Filter(ticker, userID string) ([]robot.Robot, error) {
	if userID == "" {
		if ticker == "" {
			return s.FindActivated()
		}

		return s.FindActivatedByTicker(ticker)
	}

	id, err := strconv.Atoi(userID)
	if err != nil {
		return nil, robot.ErrInvalidID
	}

	if ticker == "" {
		return s.FindActivatedByUserID(id)
	}

	return s.FindActivatedByTickerUserID(ticker, id)
}

// FavouriteRobot добавляет в базу данных нового избранного робота.
func (s *RobotStorage) FavouriteRobot(parentRobotID, userID int) error {
	r, err := s.FindByID(parentRobotID)
	if err != nil {
		return fmt.Errorf("%w: robotStorage.FindByID return with error: %s", robot.ErrNotFound, err)
	}

	r.OwnerUserID = userID
	r.ParentRobotID = parentRobotID
	r.IsFavourite = true
	r.IsActive = false
	r.DeletedAt.Valid = false
	r.DealsCount = 0
	r.FactYield = 0.0

	if err = s.Create(r); err != nil {
		return fmt.Errorf("robotStorage.Create return with error: %s", err)
	}

	return nil
}

const activateRobotQuery = `UPDATE robots SET is_active = $1, activated_at = $2 WHERE robot_id = $3`

// ActivateRobot активирует робота.
func (s *RobotStorage) ActivateRobot(robotID int) error {
	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("can't start a transaction: %s", err)
	}

	if _, err = tx.Stmt(s.activateRobotStmt).Exec(true, time.Now(), robotID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("can't activate robot: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in robotStorage: %s", err)
	}

	return nil
}

const deactivateRobotQuery = `UPDATE robots SET is_active = $1, deactivated_at = $2 WHERE robot_id = $3`

// DeactivateRobot активирует робота.
func (s *RobotStorage) DeactivateRobot(robotID int) error {
	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("can't start a transaction: %s", err)
	}

	if _, err = tx.Stmt(s.deactivateRobotStmt).Exec(false, time.Now(), robotID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("can't activate robot: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in robotStorage: %s", err)
	}

	return nil
}

const findToTradingQuery = `SELECT ` + robotFieldsSelect + ` FROM robots WHERE deleted_at IS NULL AND ` +
	`((plan_start IS NOT NULL AND plan_end IS NOT NULL AND plan_start < now() AND plan_end > now()) OR is_active = true)`

// FindToTrading находит роботов которых можно запустить на торговлю
func (s *RobotStorage) FindToTrading() ([]robot.Robot, error) {
	rows, err := s.findToTradingStmt.Query()
	if err != nil {
		return nil, fmt.Errorf("can't exec query: %s", err)
	}
	defer rows.Close()

	robots, err := scanRobots(rows)
	if err != nil {
		return nil, fmt.Errorf("can't scan robots: %s", err)
	}

	return robots, nil
}

const tradeQuery = `UPDATE robots SET is_buying = $1, deals_count = $2, fact_yield = $3 WHERE robot_id = $4`

// Trade Пишет в базу изменения рбота после транзакции
func (s *RobotStorage) Trade(rob *robot.Robot) error {
	tx, err := s.db.Session.Begin()
	if err != nil {
		return fmt.Errorf("can't start a transaction: %s", err)
	}

	if _, err = tx.Stmt(s.tradeStmt).Exec(rob.IsBuying, rob.DealsCount, rob.FactYield, rob.RobotID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("can't execute trade: %s", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("can't commit in robotStorage: %s", err)
	}

	return nil
}
