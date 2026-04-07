package option

// Option описывает функциональную опцию для настройки значения типа T.
type Option[T any] func(*T)

// Apply применяет опции к значению.
func Apply[T any](dst *T, opts ...Option[T]) {
	for _, opt := range opts {
		if opt != nil {
			opt(dst)
		}
	}
}
