package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/peterbourgon/ff/v3"

	"github.com/xinau/todoistbackup/internal/client"
	"github.com/xinau/todoistbackup/internal/store"
)

type manager struct {
	client *client.Client
	store  *store.Store
	wg     sync.WaitGroup
}

func (m *manager) download(ctx context.Context, backup *client.Backup) {
	reader, _, err := m.client.DownloadBackup(ctx, backup)
	if err != nil {
		log.Fatalf("fatal: downloading backup %q from todoist: %s", backup.Version, err)
	}
	defer reader.Close()

	err = m.store.PutBackup(ctx, backup, reader)
	if err != nil {
		log.Printf("error: writting backup %q to storage: %s", backup.Version, err)
	}
	log.Printf("info: written backup %q to storage", backup.Version)
}

func (m *manager) run(ctx context.Context) {
	backups, _, err := m.client.ListBackups(ctx)
	if err != nil {
		log.Fatalf("fatal: listing todoist backups: %s", err)
	}
	log.Printf("info: found %d potentially new backups", len(backups))

	existing, err := m.store.ListVersions(ctx)
	if err != nil {
		log.Fatalf("fatal: listing backups in storage: %s", err)
	}

	log.Printf("info: starting download of missing backups")
	for _, backup := range backups {
		if _, ok := existing[backup.Version]; ok {
			continue
		}

		m.wg.Add(1)
		go func(backup *client.Backup) {
			defer m.wg.Done()
			m.download(ctx, backup)
		}(backup)
	}
	m.wg.Wait()

	versions, err := m.store.ListVersions(ctx)
	if err != nil {
		log.Fatal("fatal: listing backups in storage")
	}
	log.Printf("info: added %d new backups to storage", len(versions)-len(existing))
}

type config struct {
	client client.Config
	store  store.Config
}

func parse() (*config, error) {
	var cfg config

	fs := flag.NewFlagSet("todoistbackup", flag.ContinueOnError)
	fs.String("config.file", "config.json",
		"configuration file loaded, also TODOISTBACKUP_CONFIG_FILE")

	fs.StringVar(&cfg.client.Token, "client.token", "",
		"todoist client api integration token, also TODOISTBACKUP_CLIENT_TOKEN")
	fs.IntVar(&cfg.client.Timeout, "client.timeout", 5,
		"todoist client timeout in seconds, also TODOISTBACKUP_CLIENT_TIMEOUT")

	fs.StringVar(&cfg.store.Bucket, "store.bucket", "",
		"todoist store s3 bucket name, also TODOISTBACKUP_STORE_BUCKET")
	fs.StringVar(&cfg.store.Endpoint, "store.endpoint", "",
		"todoist store s3 endpoint address, also TODOISTBACKUP_STORE_ENDPOINT")
	fs.StringVar(&cfg.store.Region, "store.region", "",
		"todoist store s3 region, also TODOISTBACKUP_STORE_REGION")

	fs.StringVar(&cfg.store.AccessKey, "store.access_key", "",
		"todoist store s3 access key, also TODOISTBACKUP_STORE_ACCESS_KEY")
	fs.StringVar(&cfg.store.SecretKey, "store.secret_key", "",
		"todoist store s3 secret key, also TODOISTBACKUP_STORE_SECRET_KEY")

	fs.BoolVar(&cfg.store.Insecure, "store.insecure", false,
		"todoist store s3 connection insecure, also TODOISTBACKUP_STORE_INSECURE (default false)")

	err := ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("TODOISTBACKUP"),
		ff.WithConfigFileFlag("config.file"),
		ff.WithConfigFileParser(ff.JSONParser),
	)

	if err != nil {
		return nil, err
	}

	if err := cfg.client.Validate(); err != nil {
		return nil, fmt.Errorf("validating client config: %w", err)
	}

	if err := cfg.store.Validate(); err != nil {
		return nil, fmt.Errorf("validating store config: %w", err)
	}

	return &cfg, nil
}

func main() {
	cfg, err := parse()
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	} else if err != nil {
		log.Fatalf("fatal: loading configuration: %s", err)
	}

	var mgr manager
	mgr.client, err = client.NewClient(&cfg.client)
	if err != nil {
		log.Fatalf("fatal: initializing client: %s", err)
	}

	mgr.store, err = store.NewStore(&cfg.store)
	if err != nil {
		log.Fatalf("fatal: initializing storage: %s", err)
	}

	ctx := context.Background()
	mgr.run(ctx)
	os.Exit(0)
}
