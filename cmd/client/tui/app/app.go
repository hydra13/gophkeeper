package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/buildinfo"
	"github.com/hydra13/gophkeeper/pkg/clientcore"
	"github.com/hydra13/gophkeeper/pkg/clientui"
	"github.com/rivo/tview"
)

const binaryChunkSize int64 = 64 * 1024

var filterOptions = []string{"all", "login", "text", "binary", "card"}

type App struct {
	core        *clientcore.ClientCore
	application *tview.Application
	pages       *tview.Pages
	state       state

	recordList *tview.List
	detailView *tview.TextView
	statusView *tview.TextView
	filterDrop *tview.DropDown
}

func New(core *clientcore.ClientCore) *App {
	pages := tview.NewPages()
	application := tview.NewApplication()
	application.SetRoot(pages, true)
	application.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			application.Stop()
			return nil
		}
		return event
	})

	return &App{
		core:        core,
		application: application,
		pages:       pages,
	}
}

func (a *App) Run() error {
	if a.core.IsAuthenticated() {
		a.showMainScreen()
		a.setStatus("restored cached session")
		if err := a.refreshRecords(); err != nil {
			a.showError(err)
		}
	} else {
		a.showStartScreen()
	}
	return a.application.Run()
}

func (a *App) context(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func (a *App) setRootPage(name string, primitive tview.Primitive) {
	for _, page := range []string{"start", "login", "register", "main"} {
		a.pages.RemovePage(page)
	}
	a.pages.AddPage(name, primitive, true, true)
}

func (a *App) setStatus(text string) {
	if a.statusView != nil {
		a.statusView.SetText(text)
	}
}

func (a *App) showError(err error) {
	if err == nil {
		return
	}
	a.setStatus("error: " + err.Error())
	modal := tview.NewModal().
		SetText(err.Error()).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("modal")
		})
	a.pages.AddPage("modal", centered(modal, 70, 10), true, true)
}

func (a *App) refreshRecords() error {
	ctx, cancel := a.context(30 * time.Second)
	defer cancel()

	records, err := a.core.ListRecords(ctx, a.state.filter)
	if err != nil {
		return err
	}

	a.state.records = records
	a.reloadRecordList()
	return nil
}

func (a *App) reloadRecordList() {
	if a.recordList == nil {
		return
	}

	a.recordList.Clear()
	if len(a.state.records) == 0 {
		a.state.selectedID = 0
		a.detailView.SetText("No records yet.\n\nUse 'a' to add a new record or 'l' to refresh.")
		return
	}

	selectedIndex := 0
	for i, rec := range a.state.records {
		main, secondary := clientui.FormatRecordListItem(rec)
		recordID := rec.ID
		a.recordList.AddItem(main, secondary, 0, func() {
			a.state.selectedID = recordID
			a.updateDetail()
		})
		if rec.ID == a.state.selectedID {
			selectedIndex = i
		}
	}

	if a.state.selectedID == 0 {
		a.state.selectedID = a.state.records[0].ID
		selectedIndex = 0
	}

	a.recordList.SetCurrentItem(selectedIndex)
	a.updateDetail()
}

func (a *App) selectedRecord() *models.Record {
	for i := range a.state.records {
		if a.state.records[i].ID == a.state.selectedID {
			return &a.state.records[i]
		}
	}
	return nil
}

func (a *App) updateDetail() {
	rec := a.selectedRecord()
	if rec == nil {
		a.detailView.SetText("Select a record to inspect it.")
		return
	}
	a.detailView.SetText(clientui.FormatRecord(rec))
}

func (a *App) showStartScreen() {
	list := tview.NewList().
		ShowSecondaryText(true).
		AddItem("Login", "Open the login form", 'l', func() { a.showLoginForm("") }).
		AddItem("Register", "Create a new account", 'r', func() { a.showRegisterForm("") }).
		AddItem("Exit", "Close the client", 'q', func() { a.application.Stop() })

	list.SetBorder(true).SetTitle(fmt.Sprintf(" GophKeeper TUI %s ", buildinfo.Short()))

	footer := tview.NewTextView()
	footer.SetText("Navigate with arrows or Tab. Press Enter to choose. Ctrl+C also exits.")
	footer.SetDynamicColors(true)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(centered(list, 70, 12), 12, 0, true).
		AddItem(footer, 2, 0, false)

	a.setRootPage("start", root)
	a.application.SetFocus(list)
}

func (a *App) showLoginForm(prefillEmail string) {
	emailField := tview.NewInputField().SetLabel("Email").SetText(prefillEmail)
	passwordField := tview.NewInputField().SetLabel("Password").SetMaskCharacter('*')

	form := tview.NewForm().
		AddFormItem(emailField).
		AddFormItem(passwordField).
		AddButton("Login", func() {
			if err := a.login(emailField.GetText(), passwordField.GetText()); err != nil {
				a.showError(err)
				return
			}
			a.showMainScreen()
			if err := a.refreshRecords(); err != nil {
				a.showError(err)
			}
		}).
		AddButton("Back", func() {
			a.showStartScreen()
		})

	form.SetBorder(true).SetTitle(" Login ")
	a.setRootPage("login", centered(form, 70, 12))
	a.application.SetFocus(form)
}

func (a *App) showRegisterForm(prefillEmail string) {
	emailField := tview.NewInputField().SetLabel("Email").SetText(prefillEmail)
	passwordField := tview.NewInputField().SetLabel("Password").SetMaskCharacter('*')

	form := tview.NewForm().
		AddFormItem(emailField).
		AddFormItem(passwordField).
		AddButton("Register", func() {
			email := emailField.GetText()
			password := passwordField.GetText()

			ctx, cancel := a.context(10 * time.Second)
			defer cancel()

			if err := a.core.Register(ctx, email, password); err != nil {
				a.showError(err)
				return
			}

			a.showRegisterSuccess(email, password)
		}).
		AddButton("Back", func() {
			a.showStartScreen()
		})

	form.SetBorder(true).SetTitle(" Register ")
	a.setRootPage("register", centered(form, 70, 12))
	a.application.SetFocus(form)
}

func (a *App) showRegisterSuccess(email, password string) {
	modal := tview.NewModal().
		SetText("registered successfully").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("modal")
			if err := a.login(email, password); err != nil {
				a.showLoginForm(email)
				a.showError(fmt.Errorf("autologin after register: %w", err))
				return
			}
			a.showMainScreen()
			if err := a.refreshRecords(); err != nil {
				a.showError(err)
			}
		})
	a.pages.AddPage("modal", centered(modal, 50, 10), true, true)
}

func (a *App) login(email, password string) error {
	ctx, cancel := a.context(10 * time.Second)
	defer cancel()

	if err := a.core.Login(ctx, email, password); err != nil {
		return err
	}

	a.setStatus("logged in successfully")
	return nil
}

func (a *App) showMainScreen() {
	a.recordList = tview.NewList().ShowSecondaryText(true)
	a.recordList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(a.state.records) {
			a.state.selectedID = a.state.records[index].ID
			a.updateDetail()
		}
	})

	a.detailView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	a.detailView.SetBorder(true).SetTitle(" Details ")

	a.statusView = tview.NewTextView().SetDynamicColors(true)
	a.statusView.SetBorder(true).SetTitle(" Status ")

	a.filterDrop = tview.NewDropDown().
		SetLabel("Filter: ").
		SetOptions(filterOptions, func(option string, index int) {
			if option == "all" {
				a.state.filter = ""
			} else {
				a.state.filter = models.RecordType(option)
			}
			if err := a.refreshRecords(); err != nil {
				a.showError(err)
			}
		})

	toolbar := tview.NewFlex().
		AddItem(a.filterDrop, 30, 0, false).
		AddItem(tview.NewTextView().
			SetText("Keys: l list  g get/save  a add  u update  d delete  s sync  o logout  q exit"), 0, 1, false)
	toolbar.SetBorder(true).SetTitle(" Actions ")

	a.recordList.SetBorder(true).SetTitle(" Records ")
	content := tview.NewFlex().
		AddItem(a.recordList, 0, 2, true).
		AddItem(a.detailView, 0, 3, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(toolbar, 3, 0, false).
		AddItem(content, 0, 1, true).
		AddItem(a.statusView, 3, 0, false)

	root.SetInputCapture(a.handleMainInput)
	a.setRootPage("main", root)
	a.application.SetFocus(a.recordList)
}

func (a *App) handleMainInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'q':
		a.application.Stop()
		return nil
	case 'l':
		if err := a.refreshRecords(); err != nil {
			a.showError(err)
			return nil
		}
		a.setStatus("records refreshed")
		return nil
	case 'g':
		a.handleGet()
		return nil
	case 'a':
		a.showAddTypeDialog()
		return nil
	case 'u':
		a.showUpdateDialog()
		return nil
	case 'd':
		a.showDeleteDialog()
		return nil
	case 's':
		a.handleSync()
		return nil
	case 'o':
		a.handleLogout()
		return nil
	}
	return event
}

func centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false),
			width, 1, true,
		).
		AddItem(nil, 0, 1, false)
}
