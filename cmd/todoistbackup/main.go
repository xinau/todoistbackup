package main

import (
	"context"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/xinau/todoistbackup/internal/client"
	"github.com/xinau/todoistbackup/internal/config"
	"github.com/xinau/todoistbackup/internal/store"
)

var (
	configF = flag.String("config.file", "config.json", "configuration file to load")
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

func main() {
	ctx := context.Background()
	flag.Parse()

	var mgr manager

	config, err := config.Load(*configF)
	if err != nil {
		log.Fatalf("fatal: loading configuration %s: %s", *configF, err)
	}

	mgr.client, err = client.NewClient(config.Client)
	if err != nil {
		log.Fatalf("fatal: initializing client: %s", err)
	}

	mgr.store, err = store.NewStore(config.Store)
	if err != nil {
		log.Fatalf("fatal: initializing storage: %s", err)
	}

	mgr.run(ctx)
	os.Exit(0)
}
