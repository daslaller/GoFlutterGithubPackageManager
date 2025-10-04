// Package components provides reusable view components for the TUI application
// following the bubbles view component pattern for better modularity and state management.

package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/daslaller/GoFlutterGithubPackageManager/internal/core"
)

// ViewComponent represents a reusable view component that can handle its own state
type ViewComponent interface {
	tea.Model

	// SetData allows the parent to pass data to the component
	SetData(data interface{})

	// GetResult returns any result data from the component
	GetResult() interface{}

	// IsComplete returns whether the component has finished its task
	IsComplete() bool
}

// ViewManager manages multiple view components and their transitions
type ViewManager struct {
	currentView ViewComponent
	views       map[string]ViewComponent
	cfg         core.Config
	logger      *core.Logger
}

// NewViewManager creates a new view manager
func NewViewManager(cfg core.Config, logger *core.Logger) *ViewManager {
	return &ViewManager{
		views:  make(map[string]ViewComponent),
		cfg:    cfg,
		logger: logger,
	}
}

// RegisterView registers a view component with a name
func (vm *ViewManager) RegisterView(name string, view ViewComponent) {
	vm.views[name] = view
}

// SwitchTo switches to a named view component
func (vm *ViewManager) SwitchTo(name string, data interface{}) error {
	if view, exists := vm.views[name]; exists {
		view.SetData(data)
		vm.currentView = view
		return nil
	}
	return core.ErrViewNotFound{Name: name}
}

// Update delegates to the current view
func (vm *ViewManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if vm.currentView == nil {
		return vm, nil
	}

	updatedView, cmd := vm.currentView.Update(msg)
	vm.currentView = updatedView.(ViewComponent)
	return vm, cmd
}

// View delegates to the current view
func (vm *ViewManager) View() string {
	if vm.currentView == nil {
		return "No view selected"
	}
	return vm.currentView.View()
}

// Init delegates to the current view
func (vm *ViewManager) Init() tea.Cmd {
	if vm.currentView == nil {
		return nil
	}
	return vm.currentView.Init()
}

// GetCurrentView returns the current view component
func (vm *ViewManager) GetCurrentView() ViewComponent {
	return vm.currentView
}
