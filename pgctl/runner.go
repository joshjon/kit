package pgctl

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cohesivestack/valgo"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/urfave/cli/v2"

	"github.com/joshjon/kit/log"
	"github.com/joshjon/kit/pgdb"
)

const (
	defaultPort = 5432
	defaultDB   = "postgres"
)

type RunnerConfig struct {
	DBName     string // required
	Migrations fs.FS  // required
	Logger     log.Logger
}

type Runner struct {
	dbName     string
	migrations fs.FS
	logger     log.Logger
}

func NewRunner(cfg RunnerConfig) (*Runner, error) {
	if cfg.DBName == "" {
		return nil, errors.New("db name config is required")
	}
	if cfg.Migrations == nil {
		return nil, errors.New("migrations config is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = log.NewLogger(log.WithDevelopment()).With("database", cfg.DBName)
	}
	return &Runner{
		dbName:     cfg.DBName,
		migrations: cfg.Migrations,
		logger:     cfg.Logger,
	}, nil
}

func (r *Runner) Run(args []string) error {
	app := cli.NewApp()
	app.Name = "pgctl"
	app.Usage = fmt.Sprintf("Postgres command line tool to manage the '%s' database", r.dbName)

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "host",
			Aliases: []string{"ho"},
			Value:   "127.0.0.1",
			Usage:   "[required] hostname or ip address of postgres",
			EnvVars: []string{"POSTGRES_HOST"},
		},
		&cli.IntFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Value:   defaultPort,
			Usage:   "[required] port of postgres",
			EnvVars: []string{"POSTGRES_PORT"},
		},
		&cli.StringFlag{
			Name:    "user",
			Aliases: []string{"u"},
			Value:   "",
			Usage:   "[required] username for auth when connecting to postgres",
			EnvVars: []string{"POSTGRES_USER"},
		},
		&cli.StringFlag{
			Name:    "password",
			Aliases: []string{"pw"},
			Value:   "",
			Usage:   "[required] password for auth when connecting to postgres",
			EnvVars: []string{"POSTGRES_PASSWORD"},
		},
	}

	app.Commands = []*cli.Command{
		{
			Name:  "create",
			Usage: "creates the database",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "default-db",
					Value: defaultDB,
					Usage: "default db to connect to (not '" + r.dbName + "')",
				},
			},
			Action: execCmd(r.create),
		},
		{
			Name:  "drop",
			Usage: "drops the database",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "default-db",
					Aliases: []string{"d"},
					Value:   defaultDB,
					Usage:   "default db to connect to (not '" + r.dbName + "')",
				},
			},
			Action: execCmd(r.drop),
		},
		{
			Name:   "migrate",
			Usage:  "applies all pending database schema migrations",
			Action: execCmd(r.migrate),
		},
		{
			Name:  "migrate-version",
			Usage: "migrates the database to a specific schema version",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "version",
					Aliases: []string{"v"},
					Usage:   "desired schema version",
				},
			},
			Action: execCmd(r.migrateVersion),
		},
		{
			Name:  "init",
			Usage: "creates the database and migrates to the latest schema version",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "default-db",
					Value: defaultDB,
					Usage: "default db to connect to (not the coro db to be created)",
				},
			},
			Action: execCmd(r.init),
		},
	}

	return app.Run(args)
}

func (r *Runner) create(ctx context.Context, cfg config, c *cli.Context) error {
	database := c.String("default-db")
	exitOnInvalidFlags(c, valgo.Is(valgo.String(database, "default-db").Not().Blank()))

	r.logger.Info("connecting to default database", "database", database)
	hostPort := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	conn, err := pgdb.Dial(ctx, cfg.user, cfg.password, hostPort, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	r.logger.Info("creating database")
	if _, err = conn.Exec(ctx, "CREATE DATABASE "+sanitize(r.dbName)); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code != pgerrcode.DuplicateDatabase {
				return err
			}
			r.logger.Info("database already exists")
		}
	}
	r.logger.Info("database successfully created")

	return nil
}

func (r *Runner) drop(ctx context.Context, cfg config, c *cli.Context) error {
	database := c.String("default-db")
	exitOnInvalidFlags(c, valgo.Is(valgo.String(database, "default-db").Not().Blank()))

	r.logger.Info("connecting to default database", "database", database)
	hostPort := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	conn, err := pgdb.Dial(ctx, cfg.user, cfg.password, hostPort, database)
	if err != nil {
		return err
	}
	defer conn.Close()

	r.logger.Info("dropping database")
	if _, err = conn.Exec(ctx, "DROP DATABASE IF EXISTS "+sanitize(r.dbName)); err != nil {
		return err
	}
	r.logger.Info("database successfully dropped")

	return nil
}

func (r *Runner) migrate(ctx context.Context, cfg config, _ *cli.Context) error {
	r.logger.Info("connecting to database")
	hostPort := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	conn, err := pgdb.Dial(ctx, cfg.user, cfg.password, hostPort, r.dbName)
	if err != nil {
		return err
	}
	defer conn.Close()

	r.logger.Info("migrating database")
	if err = pgdb.Migrate(conn, r.migrations); err != nil {
		return err
	}
	r.logger.Info("successfully migrated database")

	return nil
}

func (r *Runner) migrateVersion(ctx context.Context, cfg config, c *cli.Context) error {
	version := c.Uint("version")
	exitOnInvalidFlags(c, valgo.Is(valgo.Uint64(uint64(version), "version").GreaterThan(0)))

	l := r.logger

	l.Info("connecting to database")
	hostPort := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	conn, err := pgdb.Dial(ctx, cfg.user, cfg.password, hostPort, r.dbName)
	if err != nil {
		return err
	}
	defer conn.Close()

	l = l.With("version", version)
	l.Info("migrating database")
	if err = pgdb.Migrate(conn, r.migrations, pgdb.WithVersion(version)); err != nil {
		return err
	}
	l.Info("successfully migrated database")

	return nil
}

func (r *Runner) init(ctx context.Context, cfg config, c *cli.Context) error {
	if err := r.create(ctx, cfg, c); err != nil {
		return err
	}
	if err := r.migrate(ctx, cfg, c); err != nil {
		return err
	}
	return nil
}

func execCmd(cmd func(ctx context.Context, cfg config, c *cli.Context) error) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		ctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
		defer tcancel()

		cfg := loadConfig(c)
		return cmd(ctx, cfg, c)
	}
}

type config struct {
	host     string
	port     int
	user     string
	password string
}

func (c config) validate() *valgo.Validation {
	return valgo.Is(
		valgo.String(c.host, "host").Not().Blank(),
		valgo.Int(c.port, "port").GreaterThan(0),
		valgo.String(c.user, "user").Not().Blank(),
		valgo.String(c.password, "password").Not().Blank(),
	)
}

func loadConfig(c *cli.Context) config {
	cfg := config{
		host:     c.String("host"),
		port:     c.Int("port"),
		user:     c.String("user"),
		password: c.String("password"),
	}
	exitOnInvalidFlags(c, cfg.validate())
	return cfg
}

func exitOnInvalidFlags(c *cli.Context, v *valgo.Validation) {
	if v.ToError() == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "Flag errors:")

	for _, verr := range v.ToError().(*valgo.Error).Errors() {
		fmt.Fprintf(os.Stderr, "  %s: %s\n", verr.Name(), strings.Join(verr.Messages(), ","))
	}

	fmt.Fprintln(os.Stdout) //nolint:errcheck
	cli.ShowAppHelpAndExit(c, 1)
}

func sanitize(s string) string {
	return pgx.Identifier{s}.Sanitize()
}
