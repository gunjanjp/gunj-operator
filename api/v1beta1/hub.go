/*
Copyright 2025 Gunjan Jain.

Licensed under the MIT License.
*/

package v1beta1

// Hub marks ObservabilityPlatform as a conversion hub.
// All other API versions will convert to and from this version.
func (*ObservabilityPlatform) Hub() {}

// Hub marks ObservabilityPlatformList as a conversion hub.
func (*ObservabilityPlatformList) Hub() {}
