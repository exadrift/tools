package display

import (
	"github.com/exadrift/go/tui"
	"github.com/exadrift/tools/kubex/internal/kubectl"
)

var (
	contexts     []string
	namespaces   []string
	curContext   string
	curNamespace string
)

func InitializeDisplay(contextMenu *tui.Menu, namespaceMenu *tui.Menu) error {
	var err error
	if curContext, err = kubectl.GetCurrentContext(); err != nil {
		return err
	}

	if curNamespace, err = kubectl.GetCurrentNamespace(curContext); err != nil {
		return err
	}

	if err = PopulateContexts(contextMenu); err != nil {
		return err
	}

	if err = PopulateNamespaces(namespaceMenu); err != nil {
		return err
	}

	return nil
}

func PopulateContexts(menu *tui.Menu) error {
	var err error
	contexts, err = kubectl.GetContexts()
	if err != nil {
		return err
	}
	menu.SetContents(contexts...)

	return nil
}

func PopulateNamespaces(menu *tui.Menu) error {
	var err error

	if curNamespace, err = kubectl.GetCurrentNamespace(curContext); err != nil {
		return err
	}

	namespaces, err = kubectl.GetNamespaces()
	if err != nil {
		return err
	}
	menu.SetContents(namespaces...)

	return nil
}

func UpdateContextSelection(selectedContext string, namespaceMenu *tui.Menu) error {
	curContext = selectedContext
	if err := kubectl.SetCurrentContext(selectedContext); err != nil {
		return err
	}

	return PopulateNamespaces(namespaceMenu)
}

func UpdateNamespaceSelection(selectedNamespace string) error {
	curNamespace = selectedNamespace
	return kubectl.SetCurrentNamespace(curNamespace)
}
