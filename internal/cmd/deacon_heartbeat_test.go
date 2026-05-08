package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steveyegge/gastown/internal/deacon"
)

func TestAgentLabelsWithHeartbeatAndIdle(t *testing.T) {
	ts := time.Unix(1710000000, 0).UTC()
	got := agentLabelsWithHeartbeatAndIdle([]string{
		"gt:agent",
		"role_type",
		"heartbeat:123",
		"idle:7",
		"backoff-until:999",
	}, ts, 0)

	want := []string{
		"gt:agent",
		"role_type",
		"backoff-until:999",
		"heartbeat:1710000000",
		"idle:0",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d labels, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("label[%d] = %q, want %q (all labels: %v)", i, got[i], want[i], got)
		}
	}
}

func TestRefreshDeaconHeartbeatSourcesUpdatesAgentBeadLabels(t *testing.T) {
	townRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(townRoot, ".beads"), 0755); err != nil {
		t.Fatalf("mkdir .beads: %v", err)
	}

	binDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "bd.log")
	unixScript := strings.ReplaceAll(`#!/bin/sh
echo "$@" >> "LOGPATH"
if [ "$1" = "show" ]; then
  printf '%s\n' '[{"id":"hq-deacon","labels":["gt:agent","heartbeat:123","idle:7","backoff-until:999"]}]'
  exit 0
fi
if [ "$1" = "update" ]; then
  exit 0
fi
exit 1
`, "LOGPATH", logPath)
	windowsScript := strings.ReplaceAll(`@echo off
echo %*>>"LOGPATH"
if "%1"=="show" (
  echo [{"id":"hq-deacon","labels":["gt:agent","heartbeat:123","idle:7","backoff-until:999"]}]
  exit /b 0
)
if "%1"=="update" exit /b 0
exit /b 1
`, "LOGPATH", logPath)
	writeBDStub(t, binDir, unixScript, windowsScript)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ts := time.Unix(1710000000, 0).UTC()
	if err := refreshDeaconHeartbeatSources(townRoot, "starting patrol", ts); err != nil {
		t.Fatalf("refreshDeaconHeartbeatSources: %v", err)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read bd log: %v", err)
	}
	log := string(logData)
	for _, want := range []string{
		"show hq-deacon --json",
		"update hq-deacon",
		"--set-labels=gt:agent",
		"--set-labels=backoff-until:999",
		"--set-labels=heartbeat:1710000000",
		"--set-labels=idle:0",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("bd log missing %q:\n%s", want, log)
		}
	}
	for _, stale := range []string{"--set-labels=heartbeat:123", "--set-labels=idle:7"} {
		if strings.Contains(log, stale) {
			t.Fatalf("bd log kept stale label %q:\n%s", stale, log)
		}
	}

	hb := deacon.ReadHeartbeat(townRoot)
	if hb == nil {
		t.Fatal("heartbeat file was not written")
	}
	if !hb.Timestamp.Equal(ts) {
		t.Fatalf("heartbeat timestamp = %s, want %s", hb.Timestamp, ts)
	}
	if hb.Cycle != 1 {
		t.Fatalf("heartbeat cycle = %d, want 1", hb.Cycle)
	}
	if hb.LastAction != "starting patrol" {
		t.Fatalf("heartbeat last action = %q, want %q", hb.LastAction, "starting patrol")
	}
}
