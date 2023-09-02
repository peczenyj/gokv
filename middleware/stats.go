package middleware

type Incrementer interface {
	Inc()
}

type Adder interface {
	Add(int64)
}

type FloatAdder interface {
	Add(float64)
}

type Observer interface {
	Observe(float64)
}

func WrapAdderToIncrementer(a Adder) Incrementer {
	return &adderToIncrementer{a}
}

type adderToIncrementer struct {
	a Adder
}

func (i *adderToIncrementer) Inc() {
	i.a.Add(1)
}

func WrapFloatAdderToIncrementer(a FloatAdder) Incrementer {
	return &floatAdderToIncrementer{a}
}

type floatAdderToIncrementer struct {
	a FloatAdder
}

func (i *floatAdderToIncrementer) Inc() {
	i.a.Add(1)
}

func WithSetHitCounter(counter Incrementer) Option {
	return func(m *middleware) {
		m.setHitCounters = append(m.setHitCounters, counter)
	}
}

func WithGetHitCounter(counter Incrementer) Option {
	return func(m *middleware) {
		m.getHitCounters = append(m.getHitCounters, counter)
	}
}

func WithGetMissCounter(counter Incrementer) Option {
	return func(m *middleware) {
		m.getMissCounters = append(m.getMissCounters, counter)
	}
}

func WithDeleteHitCounter(counter Incrementer) Option {
	return func(m *middleware) {
		m.deleteHitCounters = append(m.deleteHitCounters, counter)
	}
}

func WithSetDurationObserver(observer Observer) Option {
	return func(m *middleware) {
		m.setObservers = append(m.setObservers, observer)
	}
}

func WithGetDurationObserver(observer Observer) Option {
	return func(m *middleware) {
		m.getObservers = append(m.getObservers, observer)
	}
}

func WithDeleteDurationObserver(observer Observer) Option {
	return func(m *middleware) {
		m.deleteObservers = append(m.deleteObservers, observer)
	}
}
