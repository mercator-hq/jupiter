// Package enforcement provides enforcement actions for limit violations.
//
// # Overview
//
// The enforcement package defines what happens when limits are exceeded:
//
//   - Block: Reject the request with 429 Too Many Requests
//   - Queue: Hold the request until capacity is available
//   - Downgrade: Route to a cheaper model
//   - Alert: Trigger an alert but allow the request
//
// # Usage
//
//	enforcer := enforcement.NewEnforcer(enforcement.Config{
//	    DefaultAction: enforcement.ActionBlock,
//	    QueueDepth:    100,
//	    QueueTimeout:  30 * time.Second,
//	})
//
//	// Execute enforcement action
//	result, err := enforcer.Enforce(ctx, action, limitResult)
//
// # Thread Safety
//
// The Enforcer is thread-safe and can be used concurrently from multiple goroutines.
package enforcement
