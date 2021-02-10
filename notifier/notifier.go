package notifier

type Notifier interface {
	Notify(name string)
}