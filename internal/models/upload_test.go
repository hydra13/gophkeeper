package models

import "testing"

func TestUploadSessionIsCompleted(t *testing.T) {
	u := &UploadSession{Status: UploadStatusCompleted}
	if !u.IsCompleted() {
		t.Fatal("expected upload to be completed")
	}

	u2 := &UploadSession{Status: UploadStatusPending}
	if u2.IsCompleted() {
		t.Fatal("expected pending upload to not be completed")
	}
}

func TestUploadSessionIsAborted(t *testing.T) {
	u := &UploadSession{Status: UploadStatusAborted}
	if !u.IsAborted() {
		t.Fatal("expected upload to be aborted")
	}

	u2 := &UploadSession{Status: UploadStatusPending}
	if u2.IsAborted() {
		t.Fatal("expected pending upload to not be aborted")
	}
}

func TestUploadSessionIsResumable(t *testing.T) {
	u := &UploadSession{
		Status:         UploadStatusPending,
		TotalChunks:    5,
		ReceivedChunks: 2,
	}
	if !u.IsResumable() {
		t.Fatal("expected upload to be resumable")
	}

	// Все чанки получены
	u2 := &UploadSession{
		Status:         UploadStatusPending,
		TotalChunks:    5,
		ReceivedChunks: 5,
	}
	if u2.IsResumable() {
		t.Fatal("expected upload with all chunks to not be resumable")
	}

	// Загрузка завершена
	u3 := &UploadSession{
		Status:         UploadStatusCompleted,
		TotalChunks:    5,
		ReceivedChunks: 5,
	}
	if u3.IsResumable() {
		t.Fatal("expected completed upload to not be resumable")
	}
}

func TestUploadSessionCompleteChunk(t *testing.T) {
	u := &UploadSession{
		Status:         UploadStatusPending,
		TotalChunks:    3,
		ReceivedChunks: 0,
	}

	if err := u.CompleteChunk(0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ReceivedChunks != 1 {
		t.Fatalf("expected 1 received chunk, got %d", u.ReceivedChunks)
	}

	if err := u.CompleteChunk(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Последний чанк завершает загрузку
	if err := u.CompleteChunk(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Status != UploadStatusCompleted {
		t.Fatalf("expected status completed, got %s", u.Status)
	}
}

func TestUploadSessionCompleteChunk_Completed(t *testing.T) {
	u := &UploadSession{
		Status:         UploadStatusCompleted,
		TotalChunks:    3,
		ReceivedChunks: 3,
	}
	if err := u.CompleteChunk(0); err != ErrUploadCompleted {
		t.Fatalf("expected ErrUploadCompleted, got: %v", err)
	}
}

func TestUploadSessionCompleteChunk_Aborted(t *testing.T) {
	u := &UploadSession{
		Status:         UploadStatusAborted,
		TotalChunks:    3,
		ReceivedChunks: 0,
	}
	if err := u.CompleteChunk(0); err != ErrUploadAborted {
		t.Fatalf("expected ErrUploadAborted, got: %v", err)
	}
}

func TestUploadSessionCompleteChunk_OutOfRange(t *testing.T) {
	u := &UploadSession{
		Status:         UploadStatusPending,
		TotalChunks:    3,
		ReceivedChunks: 0,
	}
	if err := u.CompleteChunk(5); err != ErrChunkOutOfRange {
		t.Fatalf("expected ErrChunkOutOfRange, got: %v", err)
	}
	if err := u.CompleteChunk(-1); err != ErrChunkOutOfRange {
		t.Fatalf("expected ErrChunkOutOfRange for negative index, got: %v", err)
	}
}

func TestUploadSessionCompleteChunk_Duplicate(t *testing.T) {
	u := &UploadSession{
		Status:           UploadStatusPending,
		TotalChunks:      3,
		ReceivedChunks:   1,
		ReceivedChunkSet: map[int64]bool{0: true},
	}
	// Пытаемся отправить чанк 0 повторно — но порядок проверяется первым,
	// поэтому дубликат при строгом порядке = ErrChunkOutOfOrder
	if err := u.CompleteChunk(0); err != ErrChunkOutOfOrder {
		t.Fatalf("expected ErrChunkOutOfOrder for duplicate chunk (out of order), got: %v", err)
	}
	if u.ReceivedChunks != 1 {
		t.Fatalf("expected 1 received chunk after duplicate attempt, got %d", u.ReceivedChunks)
	}
}

func TestUploadSessionAbort(t *testing.T) {
	u := &UploadSession{
		Status:         UploadStatusPending,
		TotalChunks:    3,
		ReceivedChunks: 1,
	}
	if err := u.Abort(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Status != UploadStatusAborted {
		t.Fatalf("expected aborted status, got %s", u.Status)
	}
}

func TestUploadSessionAbort_NotPending(t *testing.T) {
	u := &UploadSession{Status: UploadStatusCompleted}
	if err := u.Abort(); err == nil {
		t.Fatal("expected error when aborting completed upload")
	}
}

func TestUploadSessionMissingChunks(t *testing.T) {
	u := &UploadSession{
		Status:            UploadStatusPending,
		TotalChunks:       4,
		ReceivedChunks:    0,
		ReceivedChunkSet:  map[int64]bool{0: true, 2: true},
	}
	missing := u.MissingChunks()
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing chunks, got %d", len(missing))
	}
	expected := map[int64]bool{1: true, 3: true}
	for _, idx := range missing {
		if !expected[idx] {
			t.Fatalf("unexpected missing chunk index %d", idx)
		}
	}
}

func TestUploadSessionResume(t *testing.T) {
	u := &UploadSession{
		Status:           UploadStatusPending,
		TotalChunks:      4,
		ReceivedChunks:   2,
		ReceivedChunkSet: map[int64]bool{0: true, 1: true},
	}

	// Resume: отправляем недостающие чанки
	if err := u.CompleteChunk(2); err != nil {
		t.Fatalf("chunk 2: unexpected error: %v", err)
	}
	if err := u.CompleteChunk(3); err != nil {
		t.Fatalf("chunk 3: unexpected error: %v", err)
	}
	if u.Status != UploadStatusCompleted {
		t.Fatalf("expected completed status after resume, got %s", u.Status)
	}
	if u.ReceivedChunks != 4 {
		t.Fatalf("expected 4 received chunks, got %d", u.ReceivedChunks)
	}
}

// --- DownloadSession tests ---

func TestDownloadSessionIsCompleted(t *testing.T) {
	d := &DownloadSession{Status: DownloadStatusCompleted}
	if !d.IsCompleted() {
		t.Fatal("expected download to be completed")
	}

	d2 := &DownloadSession{Status: DownloadStatusActive}
	if d2.IsCompleted() {
		t.Fatal("expected active download to not be completed")
	}
}

func TestDownloadSessionIsAborted(t *testing.T) {
	d := &DownloadSession{Status: DownloadStatusAborted}
	if !d.IsAborted() {
		t.Fatal("expected download to be aborted")
	}

	d2 := &DownloadSession{Status: DownloadStatusActive}
	if d2.IsAborted() {
		t.Fatal("expected active download to not be aborted")
	}
}

func TestDownloadSessionIsResumable(t *testing.T) {
	d := &DownloadSession{
		Status:           DownloadStatusActive,
		TotalChunks:      5,
		ConfirmedChunks:  2,
	}
	if !d.IsResumable() {
		t.Fatal("expected download to be resumable")
	}

	// Все чанки подтверждены
	d2 := &DownloadSession{
		Status:           DownloadStatusActive,
		TotalChunks:      5,
		ConfirmedChunks:  5,
	}
	if d2.IsResumable() {
		t.Fatal("expected download with all chunks confirmed to not be resumable")
	}

	// Скачивание прервано
	d3 := &DownloadSession{
		Status:          DownloadStatusAborted,
		TotalChunks:     5,
		ConfirmedChunks: 2,
	}
	if d3.IsResumable() {
		t.Fatal("expected aborted download to not be resumable")
	}
}

func TestDownloadSessionConfirmChunk(t *testing.T) {
	d := &DownloadSession{
		Status:          DownloadStatusActive,
		TotalChunks:     3,
		ConfirmedChunks: 0,
	}

	if err := d.ConfirmChunk(0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.ConfirmedChunks != 1 {
		t.Fatalf("expected 1 confirmed chunk, got %d", d.ConfirmedChunks)
	}

	if err := d.ConfirmChunk(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Последний чанк завершает скачивание
	if err := d.ConfirmChunk(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status != DownloadStatusCompleted {
		t.Fatalf("expected status completed, got %s", d.Status)
	}
}

func TestDownloadSessionConfirmChunk_Completed(t *testing.T) {
	d := &DownloadSession{
		Status:          DownloadStatusCompleted,
		TotalChunks:     3,
		ConfirmedChunks: 3,
	}
	if err := d.ConfirmChunk(0); err != ErrDownloadCompleted {
		t.Fatalf("expected ErrDownloadCompleted, got: %v", err)
	}
}

func TestDownloadSessionConfirmChunk_Aborted(t *testing.T) {
	d := &DownloadSession{
		Status:          DownloadStatusAborted,
		TotalChunks:     3,
		ConfirmedChunks: 0,
	}
	if err := d.ConfirmChunk(0); err != ErrDownloadAborted {
		t.Fatalf("expected ErrDownloadAborted, got: %v", err)
	}
}

func TestDownloadSessionConfirmChunk_OutOfRange(t *testing.T) {
	d := &DownloadSession{
		Status:          DownloadStatusActive,
		TotalChunks:     3,
		ConfirmedChunks: 0,
	}
	if err := d.ConfirmChunk(5); err != ErrChunkOutOfRange {
		t.Fatalf("expected ErrChunkOutOfRange, got: %v", err)
	}
	if err := d.ConfirmChunk(-1); err != ErrChunkOutOfRange {
		t.Fatalf("expected ErrChunkOutOfRange for negative index, got: %v", err)
	}
}

func TestDownloadSessionConfirmChunk_Duplicate(t *testing.T) {
	d := &DownloadSession{
		Status:            DownloadStatusActive,
		TotalChunks:       3,
		ConfirmedChunks:   1,
		ConfirmedChunkSet: map[int64]bool{0: true},
	}
	// Пытаемся подтвердить чанк 0 повторно — но порядок проверяется первым,
	// поэтому дубликат при строгом порядке = ErrChunkOutOfOrder
	if err := d.ConfirmChunk(0); err != ErrChunkOutOfOrder {
		t.Fatalf("expected ErrChunkOutOfOrder for duplicate confirm (out of order), got: %v", err)
	}
	if d.ConfirmedChunks != 1 {
		t.Fatalf("expected 1 confirmed chunk after duplicate, got %d", d.ConfirmedChunks)
	}
}

func TestDownloadSessionAbort(t *testing.T) {
	d := &DownloadSession{
		Status:          DownloadStatusActive,
		TotalChunks:     3,
		ConfirmedChunks: 1,
	}
	if err := d.Abort(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Status != DownloadStatusAborted {
		t.Fatalf("expected aborted status, got %s", d.Status)
	}
}

func TestDownloadSessionAbort_NotActive(t *testing.T) {
	d := &DownloadSession{Status: DownloadStatusCompleted}
	if err := d.Abort(); err != ErrDownloadNotActive {
		t.Fatalf("expected ErrDownloadNotActive, got: %v", err)
	}
}

func TestDownloadSessionRemainingChunks(t *testing.T) {
	d := &DownloadSession{
		Status:             DownloadStatusActive,
		TotalChunks:        4,
		ConfirmedChunks:    0,
		ConfirmedChunkSet:  map[int64]bool{0: true, 2: true},
	}
	remaining := d.RemainingChunks()
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining chunks, got %d", len(remaining))
	}
	expected := map[int64]bool{1: true, 3: true}
	for _, idx := range remaining {
		if !expected[idx] {
			t.Fatalf("unexpected remaining chunk index %d", idx)
		}
	}
}

func TestDownloadSessionResume(t *testing.T) {
	d := &DownloadSession{
		Status:             DownloadStatusActive,
		TotalChunks:        4,
		ConfirmedChunks:    2,
		ConfirmedChunkSet:  map[int64]bool{0: true, 1: true},
	}

	// Resume: подтверждаем недостающие чанки
	if err := d.ConfirmChunk(2); err != nil {
		t.Fatalf("confirm chunk 2: unexpected error: %v", err)
	}
	if err := d.ConfirmChunk(3); err != nil {
		t.Fatalf("confirm chunk 3: unexpected error: %v", err)
	}
	if d.Status != DownloadStatusCompleted {
		t.Fatalf("expected completed status after resume, got %s", d.Status)
	}
	if d.ConfirmedChunks != 4 {
		t.Fatalf("expected 4 confirmed chunks, got %d", d.ConfirmedChunks)
	}
}
