package k8s

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//go:generate counterfeiter sigs.k8s.io/controller-runtime/pkg/client.Client
//go:generate counterfeiter sigs.k8s.io/controller-runtime/pkg/client.StatusWriter
