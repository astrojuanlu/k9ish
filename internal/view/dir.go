// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view

import (
	"context"
	"os"
	"path"

	"github.com/derailed/k9s/internal"
	"github.com/derailed/k9s/internal/client"
	"github.com/derailed/k9s/internal/ui"
	"github.com/derailed/tcell/v2"
)

// Dir represents a command directory view.
type Dir struct {
	ResourceViewer
	path string
}

// NewDir returns a new instance.
func NewDir(s string) ResourceViewer {
	d := Dir{
		ResourceViewer: NewBrowser(client.DirGVR),
		path:           s,
	}
	d.GetTable().SetBorderFocusColor(tcell.ColorAliceBlue)
	d.GetTable().SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorAliceBlue).Attributes(tcell.AttrNone))
	d.AddBindKeysFn(d.bindKeys)
	d.SetContextFn(d.dirContext)

	return &d
}

// Init initializes the view.
func (d *Dir) Init(ctx context.Context) error {
	if err := d.ResourceViewer.Init(ctx); err != nil {
		return err
	}

	return nil
}

func (d *Dir) dirContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, internal.KeyPath, d.path)
}

func (d *Dir) bindKeys(aa *ui.KeyActions) {
	// !!BOZO!! Lame!
	aa.Delete(ui.KeyShiftA, tcell.KeyCtrlS, tcell.KeyCtrlSpace, ui.KeySpace)
	aa.Delete(tcell.KeyCtrlW, tcell.KeyCtrlL, tcell.KeyCtrlD, tcell.KeyCtrlZ)
	aa.Bulk(ui.KeyMap{
		ui.KeyY:        ui.NewKeyAction(yamlAction, d.viewCmd, true),
		tcell.KeyEnter: ui.NewKeyAction("Goto", d.gotoCmd, true),
	})
}

func (d *Dir) viewCmd(evt *tcell.EventKey) *tcell.EventKey {
	sel := d.GetTable().GetSelectedItem()
	if sel == "" {
		return evt
	}

	if path.Ext(sel) == "" {
		return nil
	}

	yaml, err := os.ReadFile(sel)
	if err != nil {
		d.App().Flash().Err(err)
		return nil
	}

	details := NewDetails(d.App(), yamlAction, sel, contentYAML, true).Update(string(yaml))
	if err := d.App().inject(details, false); err != nil {
		d.App().Flash().Err(err)
	}

	return nil
}

func isManifest(s string) bool {
	ext := path.Ext(s)
	return ext == ".yml" || ext == ".yaml"
}

func (d *Dir) gotoCmd(evt *tcell.EventKey) *tcell.EventKey {
	if d.GetTable().CmdBuff().IsActive() {
		return d.GetTable().activateCmd(evt)
	}

	sel := d.GetTable().GetSelectedItem()
	if sel == "" {
		return evt
	}

	if isManifest(sel) {
		d.App().Flash().Errf("you must select a directory")
		return nil
	}

	v := NewDir(sel)
	if err := d.App().inject(v, false); err != nil {
		d.App().Flash().Err(err)
	}

	return evt
}
