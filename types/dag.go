package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// DAG (Directed Acyclic Graph) manages components and their dependencies
type DAG struct {
	components []*Component `json:"-"` // Not exported in JSON, we export via MarshalJSON
}

// NewDAG creates a new DAG with the given components
func NewDAG(components []*Component) (*DAG, error) {
	dag := &DAG{components: components}

	// Validate no cycles
	if err := dag.validateNoCycles(); err != nil {
		return nil, err
	}

	return dag, nil
}

// NewPizzaOrderDAG creates the default pizza order component graph
func NewPizzaOrderDAG() *DAG {
	now := time.Now()

	components := []*Component{
		{
			Type:       ComponentPayment,
			State:      StateIncomplete, // First step - ready to start immediately
			DependsOn:  []ComponentType{},
			UpdateTime: now,
		},
		{
			Type:       ComponentMakeDough,
			State:      StateNeedsInit, // Waiting for payment
			DependsOn:  []ComponentType{ComponentPayment},
			UpdateTime: now,
		},
		{
			Type:       ComponentAddToppings,
			State:      StateNeedsInit, // Waiting for dough
			DependsOn:  []ComponentType{ComponentMakeDough},
			UpdateTime: now,
		},
		{
			Type:       ComponentBakePizza,
			State:      StateNeedsInit, // Waiting for toppings
			DependsOn:  []ComponentType{ComponentAddToppings},
			UpdateTime: now,
		},
		{
			Type:       ComponentDeliver,
			State:      StateNeedsInit, // Waiting for baking
			DependsOn:  []ComponentType{ComponentBakePizza},
			UpdateTime: now,
		},
	}

	dag, _ := NewDAG(components) // We know this won't error
	return dag
}

// GetComponent finds a component by type
func (d *DAG) GetComponent(componentType ComponentType) (*Component, error) {
	for _, c := range d.components {
		if c.Type == componentType {
			return c, nil
		}
	}
	return nil, fmt.Errorf("component %s not found", componentType)
}

// GetComponents returns all components (for JSON marshaling)
func (d *DAG) GetComponents() []*Component {
	return d.components
}

// CompleteComponent marks a component as completed
func (d *DAG) CompleteComponent(componentType ComponentType) error {
	component, err := d.GetComponent(componentType)
	if err != nil {
		return err
	}

	if component.State != StateIncomplete {
		return fmt.Errorf("component %s is not in INCOMPLETE state (current: %s)", componentType, component.State)
	}

	// Mark as completed
	now := time.Now()
	component.State = StateCompleted
	component.CompleteTime = &now
	component.UpdateTime = now

	// Check if any dependent components can now be started
	d.updateDependentComponents()

	return nil
}

// updateDependentComponents checks all components and moves them to INCOMPLETE if dependencies are met
func (d *DAG) updateDependentComponents() {
	for _, component := range d.components {
		if component.State != StateNeedsInit {
			continue
		}

		// Check if all dependencies are complete
		allDepsComplete := true
		for _, depType := range component.DependsOn {
			dep, err := d.GetComponent(depType)
			if err != nil || dep.State != StateCompleted {
				allDepsComplete = false
				break
			}
		}

		// If all dependencies met, move to INCOMPLETE (ready to work on)
		if allDepsComplete {
			component.State = StateIncomplete
			component.UpdateTime = time.Now()
		}
	}
}

// AllComponentsCompleted checks if all components are done
func (d *DAG) AllComponentsCompleted() bool {
	for _, c := range d.components {
		if c.State != StateCompleted {
			return false
		}
	}
	return true
}

// GetNextComponent returns the next component that can be worked on
func (d *DAG) GetNextComponent() *Component {
	for _, c := range d.components {
		if c.State == StateIncomplete {
			return c
		}
	}
	return nil
}

// Clone creates a deep copy of the DAG
func (d *DAG) Clone() *DAG {
	clonedComponents := make([]*Component, len(d.components))

	for i, c := range d.components {
		clonedDeps := make([]ComponentType, len(c.DependsOn))
		copy(clonedDeps, c.DependsOn)

		var clonedCompleteTime *time.Time
		if c.CompleteTime != nil {
			t := *c.CompleteTime
			clonedCompleteTime = &t
		}

		clonedComponents[i] = &Component{
			Type:         c.Type,
			State:        c.State,
			DependsOn:    clonedDeps,
			UpdateTime:   c.UpdateTime,
			CompleteTime: clonedCompleteTime,
		}
	}

	return &DAG{components: clonedComponents}
}

// validateNoCycles checks for circular dependencies
func (d *DAG) validateNoCycles() error {
	visited := make(map[ComponentType]bool)
	recStack := make(map[ComponentType]bool)

	for _, component := range d.components {
		if !visited[component.Type] {
			if d.hasCycle(component.Type, visited, recStack) {
				return fmt.Errorf("cycle detected in DAG")
			}
		}
	}

	return nil
}

// hasCycle uses DFS to detect cycles
func (d *DAG) hasCycle(componentType ComponentType, visited, recStack map[ComponentType]bool) bool {
	visited[componentType] = true
	recStack[componentType] = true

	component, err := d.GetComponent(componentType)
	if err != nil {
		return false
	}

	for _, depType := range component.DependsOn {
		if !visited[depType] {
			if d.hasCycle(depType, visited, recStack) {
				return true
			}
		} else if recStack[depType] {
			return true // Back edge found - cycle!
		}
	}

	recStack[componentType] = false
	return false
}

// MarshalJSON custom JSON serialization (export components array)
func (d *DAG) MarshalJSON() ([]byte, error) {
	// We just return the components array
	type Alias DAG
	//return []byte(fmt.Sprintf("%v", d.components)), nil
	return json.Marshal(d.components)
}
