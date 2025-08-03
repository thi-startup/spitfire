package driver

import (
	"fmt"
	"strings"
)

// SelectorOptions configures driver selection
type SelectorOptions struct {
	PreferredDriver string
	RequireHealthy  bool
	RequireDefault  bool
	MinPriority     Priority
}

// DefaultSelectorOptions returns sensible defaults for driver selection
func DefaultSelectorOptions() SelectorOptions {
	return SelectorOptions{
		RequireHealthy: true,
		RequireDefault: false,
		MinPriority:    Experimental,
	}
}

// Suggest returns the best driver choice along with alternatives and rejected drivers
func Suggest(options SelectorOptions) (DriverState, []DriverState, []DriverState) {
	available := Available()

	var pick DriverState
	var alternatives []DriverState
	var rejects []DriverState

	// If a specific driver is preferred, try that first
	if options.PreferredDriver != "" {
		for _, ds := range available {
			if ds.Name == options.PreferredDriver || isInSlice(options.PreferredDriver, ds.Aliases) {
				if isDriverSuitable(ds, options) {
					pick = ds
					break
				} else {
					ds.Rejection = fmt.Sprintf("Preferred driver %q is not suitable: %v", options.PreferredDriver, ds.State.Error)
					rejects = append(rejects, ds)
				}
			}
		}
	}

	// If no preferred driver or it wasn't suitable, find the best available
	if pick.Empty() {
		for _, ds := range available {
			if isDriverSuitable(ds, options) {
				if pick.Empty() || ds.Priority > pick.Priority {
					if !pick.Empty() {
						// Previous pick becomes an alternative
						pick.Rejection = fmt.Sprintf("%s is preferred", ds.Name)
						alternatives = append(alternatives, pick)
					}
					pick = ds
				} else {
					ds.Rejection = fmt.Sprintf("%s is preferred", pick.Name)
					alternatives = append(alternatives, ds)
				}
			} else {
				// Driver doesn't meet requirements
				reason := getUnsuitableReason(ds, options)
				ds.Rejection = reason
				rejects = append(rejects, ds)
			}
		}
	}

	// Move remaining drivers to appropriate lists
	for _, ds := range available {
		if ds.Name == pick.Name {
			continue
		}

		found := false
		for _, alt := range alternatives {
			if alt.Name == ds.Name {
				found = true
				break
			}
		}
		for _, rej := range rejects {
			if rej.Name == ds.Name {
				found = true
				break
			}
		}

		if !found {
			if isDriverSuitable(ds, options) {
				ds.Rejection = fmt.Sprintf("%s is preferred", pick.Name)
				alternatives = append(alternatives, ds)
			} else {
				reason := getUnsuitableReason(ds, options)
				ds.Rejection = reason
				rejects = append(rejects, ds)
			}
		}
	}

	return pick, alternatives, rejects
}

// isDriverSuitable checks if a driver meets the selection criteria
func isDriverSuitable(ds DriverState, options SelectorOptions) bool {
	if options.RequireHealthy && !ds.State.Healthy {
		return false
	}

	if !ds.State.Installed {
		return false
	}

	if options.RequireDefault && !ds.Default {
		return false
	}

	if ds.Priority < options.MinPriority {
		return false
	}

	return true
}

// getUnsuitableReason returns a human-readable reason why a driver is not suitable
func getUnsuitableReason(ds DriverState, options SelectorOptions) string {
	reasons := []string{}

	if !ds.State.Installed {
		reasons = append(reasons, "not installed")
		if ds.State.Fix != "" {
			return fmt.Sprintf("Not installed: %s", ds.State.Fix)
		}
	}

	if options.RequireHealthy && !ds.State.Healthy {
		reasons = append(reasons, "not healthy")
		if ds.State.Error != nil {
			return fmt.Sprintf("Not healthy: %v", ds.State.Error)
		}
	}

	if options.RequireDefault && !ds.Default {
		reasons = append(reasons, "not default")
	}

	if ds.Priority < options.MinPriority {
		reasons = append(reasons, fmt.Sprintf("priority too low (%v < %v)", ds.Priority, options.MinPriority))
	}

	if len(reasons) > 0 {
		return strings.Join(reasons, ", ")
	}

	return "unknown reason"
}

// DisplaySupportedDrivers returns a formatted string of supported drivers
func DisplaySupportedDrivers() string {
	available := Available()
	var drivers []string

	for _, ds := range available {
		name := ds.Name
		if ds.Priority == Experimental {
			name += " (experimental)"
		} else if ds.Priority == Deprecated {
			name += " (deprecated)"
		}
		drivers = append(drivers, name)
	}

	return strings.Join(drivers, ", ")
}

// GetDriverChoices returns formatted driver choices for CLI display
func GetDriverChoices() []DriverState {
	return Available()
}

// isInSlice checks if a string is in a slice
func isInSlice(target string, slice []string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}
