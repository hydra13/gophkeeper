package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/clientui"
	"github.com/rivo/tview"
)

func (a *App) handleGet() {
	rec := a.selectedRecord()
	if rec == nil {
		a.showError(fmt.Errorf("select a record first"))
		return
	}

	if rec.Type == models.RecordTypeBinary {
		a.showSaveBinaryDialog(rec)
		return
	}

	text := tview.NewTextView().
		SetText(clientui.FormatRecord(rec)).
		SetScrollable(true).
		SetWrap(true)
	text.SetBorder(true).SetTitle(" Record ")

	form := tview.NewForm().
		AddButton("Close", func() {
			a.pages.RemovePage("modal")
		})

	content := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, false).
		AddItem(form, 3, 0, true)

	a.pages.AddPage("modal", centered(content, 80, 20), true, true)
}

func (a *App) showSaveBinaryDialog(rec *models.Record) {
	pathField := tview.NewInputField().SetLabel("Output path")
	form := tview.NewForm().
		AddFormItem(pathField).
		AddButton("Save", func() {
			ctx, cancel := a.context(30 * time.Second)
			defer cancel()

			data, err := a.core.DownloadBinary(ctx, rec.ID, binaryChunkSize)
			if err != nil {
				a.showError(err)
				return
			}
			if err := clientui.WriteBinaryFile(pathField.GetText(), data); err != nil {
				a.showError(err)
				return
			}
			a.pages.RemovePage("modal")
			a.setStatus(fmt.Sprintf("saved %d bytes to %s", len(data), pathField.GetText()))
		}).
		AddButton("Cancel", func() {
			a.pages.RemovePage("modal")
		})
	form.SetBorder(true).SetTitle(" Save Binary ")

	a.pages.AddPage("modal", centered(form, 80, 10), true, true)
}

func (a *App) showAddTypeDialog() {
	modal := tview.NewModal().
		SetText("Choose record type").
		AddButtons([]string{"login", "text", "binary", "card", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("modal")
			if buttonLabel == "Cancel" {
				return
			}
			recordType, err := clientui.ParseRecordType(buttonLabel)
			if err != nil {
				a.showError(err)
				return
			}
			a.showRecordForm("Add", recordType, nil)
		})
	a.pages.AddPage("modal", centered(modal, 60, 12), true, true)
}

func (a *App) showUpdateDialog() {
	rec := a.selectedRecord()
	if rec == nil {
		a.showError(fmt.Errorf("select a record first"))
		return
	}
	if rec.IsDeleted() {
		a.showError(fmt.Errorf("deleted records cannot be updated"))
		return
	}
	a.showRecordForm("Update", rec.Type, rec)
}

func (a *App) showRecordForm(mode string, recordType models.RecordType, existing *models.Record) {
	var (
		nameValue     string
		metadataValue string
		loginValue    string
		passwordValue string
		contentValue  string
		numberValue   string
		holderValue   string
		expiryValue   string
		cvvValue      string
		filePathValue string
	)

	if existing != nil {
		nameValue = existing.Name
		metadataValue = existing.Metadata
		switch payload := existing.Payload.(type) {
		case models.LoginPayload:
			loginValue = payload.Login
			passwordValue = payload.Password
		case models.TextPayload:
			contentValue = payload.Content
		case models.CardPayload:
			numberValue = payload.Number
			holderValue = payload.HolderName
			expiryValue = payload.ExpiryDate
			cvvValue = payload.CVV
		}
	}

	form := tview.NewForm()

	nameField := tview.NewInputField().SetLabel("Name").SetText(nameValue)
	metadataField := tview.NewInputField().SetLabel("Metadata").SetText(metadataValue)
	form.AddFormItem(nameField)
	form.AddFormItem(metadataField)

	var binaryPathField *tview.InputField

	switch recordType {
	case models.RecordTypeLogin:
		loginField := tview.NewInputField().SetLabel("Login").SetText(loginValue)
		passwordField := tview.NewInputField().SetLabel("Password").SetText(passwordValue).SetMaskCharacter('*')
		form.AddFormItem(loginField)
		form.AddFormItem(passwordField)
	case models.RecordTypeText:
		contentField := tview.NewInputField().SetLabel("Content").SetText(contentValue)
		form.AddFormItem(contentField)
	case models.RecordTypeCard:
		numberField := tview.NewInputField().SetLabel("Number").SetText(numberValue)
		holderField := tview.NewInputField().SetLabel("Holder").SetText(holderValue)
		expiryField := tview.NewInputField().SetLabel("Expiry").SetText(expiryValue)
		cvvField := tview.NewInputField().SetLabel("CVV").SetText(cvvValue).SetMaskCharacter('*')
		form.AddFormItem(numberField)
		form.AddFormItem(holderField)
		form.AddFormItem(expiryField)
		form.AddFormItem(cvvField)
	case models.RecordTypeBinary:
		label := "File path"
		if existing != nil {
			label = "New file path"
		}
		binaryPathField = tview.NewInputField().SetLabel(label).SetText(filePathValue)
		form.AddFormItem(binaryPathField)
	}

	form.AddButton("Save", func() {
		name := nameField.GetText()
		metadata := metadataField.GetText()

		fields := clientui.PayloadFields{}
		var (
			payload        models.RecordPayload
			fileData       []byte
			err            error
			recordToUpsert *models.Record
		)

		switch recordType {
		case models.RecordTypeLogin:
			fields.Login = form.GetFormItem(2).(*tview.InputField).GetText()
			fields.Password = form.GetFormItem(3).(*tview.InputField).GetText()
		case models.RecordTypeText:
			fields.Content = form.GetFormItem(2).(*tview.InputField).GetText()
		case models.RecordTypeCard:
			fields.Number = form.GetFormItem(2).(*tview.InputField).GetText()
			fields.Holder = form.GetFormItem(3).(*tview.InputField).GetText()
			fields.Expiry = form.GetFormItem(4).(*tview.InputField).GetText()
			fields.CVV = form.GetFormItem(5).(*tview.InputField).GetText()
		case models.RecordTypeBinary:
			if binaryPathField != nil && binaryPathField.GetText() != "" {
				fileData, err = clientui.ReadBinaryFile(binaryPathField.GetText())
				if err != nil {
					a.showError(err)
					return
				}
			}
		}

		payload, err = clientui.BuildPayload(recordType, fields)
		if err != nil {
			a.showError(err)
			return
		}

		if existing != nil {
			clone := *existing
			recordToUpsert = &clone
			recordToUpsert.Name = name
			recordToUpsert.Metadata = metadata
			if recordType != models.RecordTypeBinary {
				recordToUpsert.Payload = payload
			}
			if recordType == models.RecordTypeBinary && len(fileData) > 0 {
				if recordToUpsert.PayloadVersion <= 0 {
					recordToUpsert.PayloadVersion = 1
				} else {
					recordToUpsert.PayloadVersion++
				}
			}
		} else {
			recordToUpsert = &models.Record{
				Type:     recordType,
				Name:     name,
				Metadata: metadata,
				Payload:  payload,
			}
			if recordType == models.RecordTypeBinary {
				recordToUpsert.PayloadVersion = 1
			}
		}

		ctx, cancel := a.context(30 * time.Second)
		defer cancel()

		result, err := a.core.SaveRecord(ctx, recordToUpsert)
		if err != nil {
			a.showError(err)
			return
		}

		if recordType == models.RecordTypeBinary && len(fileData) > 0 {
			if err := a.core.UploadBinary(ctx, result.ID, fileData, binaryChunkSize); err != nil {
				a.showError(fmt.Errorf("upload binary: %w", err))
				return
			}
		}

		a.pages.RemovePage("modal")
		a.state.selectedID = result.ID
		if err := a.refreshRecords(); err != nil {
			a.showError(err)
			return
		}
		a.setStatus(fmt.Sprintf("%s complete: id=%d rev=%d", strings.ToLower(mode), result.ID, result.Revision))
	}).
		AddButton("Cancel", func() {
			a.pages.RemovePage("modal")
		})

	form.SetBorder(true).SetTitle(fmt.Sprintf(" %s %s ", mode, recordType))
	a.pages.AddPage("modal", centered(form, 90, 18), true, true)
}

func (a *App) showDeleteDialog() {
	rec := a.selectedRecord()
	if rec == nil {
		a.showError(fmt.Errorf("select a record first"))
		return
	}

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete %q?", rec.Name)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("modal")
			if buttonLabel != "Delete" {
				return
			}
			ctx, cancel := a.context(10 * time.Second)
			defer cancel()

			if err := a.core.DeleteRecord(ctx, rec.ID); err != nil {
				a.showError(err)
				return
			}
			a.state.selectedID = 0
			if err := a.refreshRecords(); err != nil {
				a.showError(err)
				return
			}
			a.setStatus("deleted")
		})
	a.pages.AddPage("modal", centered(modal, 60, 10), true, true)
}

func (a *App) handleSync() {
	ctx, cancel := a.context(30 * time.Second)
	defer cancel()

	if err := a.core.SyncNow(ctx); err != nil {
		a.showError(err)
		return
	}
	if err := a.refreshRecords(); err != nil {
		a.showError(err)
		return
	}
	a.setStatus("synced")
}

func (a *App) handleLogout() {
	ctx, cancel := a.context(10 * time.Second)
	defer cancel()

	if err := a.core.Logout(ctx); err != nil {
		a.showError(err)
		return
	}

	a.state = state{}
	a.showStartScreen()
}
