package display

import (
	"github.com/exadrift/tools/kubex/internal/kubectl"
	"github.com/rivo/tview"
)

var (
	contexts     []string
	namespaces   []string
	curContext   string
	curNamespace string
)

func InitializeDisplay(contextTable *tview.Table, namespaceTable *tview.Table) error {
	var err error
	if curContext, err = kubectl.GetCurrentContext(); err != nil {
		return err
	}

	if curNamespace, err = kubectl.GetCurrentNamespace(curContext); err != nil {
		return err
	}

	if err = PopulateContexts(contextTable); err != nil {
		return err
	}

	if err = PopulateNamespaces(namespaceTable); err != nil {
		return err
	}

	return nil
}

func PopulateContexts(table *tview.Table) error {
	var err error
	table.Clear()
	contexts, err = kubectl.GetContexts()
	if err != nil {
		return err
	}

	for i, name := range contexts {
		table.SetCell(i, 0, tview.NewTableCell(name))
		if name == curContext {
			table.Select(i, 0)
		}
	}

	return nil
}

func PopulateNamespaces(table *tview.Table) error {
	var err error

	if curNamespace, err = kubectl.GetCurrentNamespace(curContext); err != nil {
		return err
	}

	table.Clear()
	namespaces, err = kubectl.GetNamespaces()
	if err != nil {
		return err
	}

	for i, name := range namespaces {
		table.SetCell(i, 0, tview.NewTableCell(name))
		if name == curNamespace {
			table.Select(i, 0)
		}
	}

	return nil
}

func UpdateContextSelection(index int, namespaceTable *tview.Table) error {
	curContext = contexts[index]
	if err := kubectl.SetCurrentContext(curContext); err != nil {
		return err
	}

	return PopulateNamespaces(namespaceTable)
}

func UpdateNamespaceSelection(index int) error {
	curNamespace = namespaces[index]
	return kubectl.SetCurrentNamespace(curNamespace)
}
