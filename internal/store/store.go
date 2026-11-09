// Package store: thread-safer In-Memory-Store hinter einem Interface.
package store

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"verleih/internal/domain"
)

var (
	ErrNotFound = errors.New("nicht gefunden")
	ErrConflict = errors.New("Konflikt")
)

const maxHours = 24 * 365

type Stats struct {
	Available    int     `json:"available"`
	Rented       int     `json:"rented"`
	Total        int     `json:"total"`
	OutOfService int     `json:"outOfService"`
	Persons      int     `json:"persons"`
	OverdueCount int     `json:"overdue"`
	Revenue      float64 `json:"revenue"`
}

type Store interface {
	Items() []domain.Item
	Item(nr string) (domain.Item, error)
	AddItem(domain.Item) error
	DeleteItem(nr string) error
	ReviewDone(nr string) error
	Persons() []domain.Person
	Person(id int) (domain.Person, error)
	AddPerson(domain.Person) (domain.Person, error)
	DeletePerson(id int) error
	Rentals() []domain.Rental
	Quote(itemNr string, personID int, start time.Time, hours int, code string) (domain.Quote, error)
	CreateRental(itemNr string, personID int, start time.Time, hours int, code string) (domain.Rental, error)
	ReturnRental(id int, report *domain.DamageReport) (domain.Rental, error)
	RentedItemNrs() map[string]bool
	Stats() Stats
}

type MemoryStore struct {
	mu       sync.RWMutex
	items    map[string]domain.Item
	persons  map[int]domain.Person
	rentals  map[int]domain.Rental
	nextPers int
	nextRent int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items:    map[string]domain.Item{},
		persons:  map[int]domain.Person{},
		rentals:  map[int]domain.Rental{},
		nextPers: 1,
		nextRent: 1,
	}
}

func (s *MemoryStore) Items() []domain.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Item, 0, len(s.items))
	for _, i := range s.items {
		out = append(out, i)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Nr < out[b].Nr })
	return out
}

func (s *MemoryStore) Item(nr string) (domain.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i, ok := s.items[nr]
	if !ok {
		return i, ErrNotFound
	}
	return i, nil
}

func (s *MemoryStore) AddItem(i domain.Item) error {
	if err := i.Validate(); err != nil {
		return fmt.Errorf("ungültiger Eintrag: %w", err)
	}
	i.OutOfService, i.Damage = false, nil
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[i.Nr]; ok {
		return fmt.Errorf("Nr %s existiert bereits: %w", i.Nr, ErrConflict)
	}
	s.items[i.Nr] = i
	return nil
}

func (s *MemoryStore) rentedLocked() map[string]bool {
	m := map[string]bool{}
	for _, r := range s.rentals {
		if r.Active() {
			m[r.ItemNr] = true
		}
	}
	return m
}

func (s *MemoryStore) RentedItemNrs() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rentedLocked()
}

func (s *MemoryStore) DeleteItem(nr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[nr]; !ok {
		return ErrNotFound
	}
	if s.rentedLocked()[nr] {
		return fmt.Errorf("%s ist noch ausgeliehen: %w", nr, ErrConflict)
	}
	delete(s.items, nr)
	return nil
}

// ReviewDone beendet die Prüfung: der Artikel ist wieder im Betrieb.
func (s *MemoryStore) ReviewDone(nr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := s.items[nr]
	if !ok {
		return ErrNotFound
	}
	if !i.OutOfService {
		return fmt.Errorf("%s ist nicht außer Betrieb: %w", nr, ErrConflict)
	}
	i.OutOfService, i.Damage = false, nil
	s.items[nr] = i
	return nil
}

func (s *MemoryStore) Persons() []domain.Person {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Person, 0, len(s.persons))
	for _, p := range s.persons {
		out = append(out, p)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	return out
}

func (s *MemoryStore) Person(id int) (domain.Person, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.persons[id]
	if !ok {
		return p, ErrNotFound
	}
	return p, nil
}

func (s *MemoryStore) AddPerson(p domain.Person) (domain.Person, error) {
	if err := p.Validate(); err != nil {
		return p, fmt.Errorf("ungültige Person: %w", err)
	}
	p.HasScan = len(p.ScanData) > 0
	p.LoyaltyBookings = 0
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextPers
	s.nextPers++
	s.persons[p.ID] = p
	return p, nil
}

func (s *MemoryStore) DeletePerson(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.persons[id]; !ok {
		return ErrNotFound
	}
	for _, r := range s.rentals {
		if r.Active() && r.PersonID == id {
			return fmt.Errorf("Person hat noch eine aktive Ausleihe: %w", ErrConflict)
		}
	}
	delete(s.persons, id)
	return nil
}

func (s *MemoryStore) Rentals() []domain.Rental {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Rental, 0, len(s.rentals))
	for _, r := range s.rentals {
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID > out[b].ID })
	return out
}

// quoteLocked prüft die Buchung und berechnet den Preis (Aufrufer hält mindestens das Read-Lock).
func (s *MemoryStore) quoteLocked(itemNr string, personID int, start time.Time, hours int, code string) (domain.Quote, domain.Item, domain.Person, error) {
	var none domain.Quote
	if hours < 1 || hours > maxHours {
		return none, domain.Item{}, domain.Person{}, fmt.Errorf("Dauer muss zwischen 1 und %d Stunden liegen", maxHours)
	}
	item, ok := s.items[itemNr]
	if !ok {
		return none, item, domain.Person{}, fmt.Errorf("%s: %w", itemNr, ErrNotFound)
	}
	p, ok := s.persons[personID]
	if !ok {
		return none, item, p, fmt.Errorf("Person %d: %w", personID, ErrNotFound)
	}
	if item.OutOfService {
		return none, item, p, fmt.Errorf("%s ist außer Betrieb: %w", itemNr, ErrConflict)
	}
	rented := s.rentedLocked()
	if rented[itemNr] {
		return none, item, p, fmt.Errorf("Schon gebucht: %s ist bereits ausgeliehen: %w", itemNr, ErrConflict)
	}
	inService, prior := 0, 0
	for _, it := range s.items {
		if !it.OutOfService {
			inService++
		}
	}
	for _, r := range s.rentals {
		if r.PersonID == personID {
			prior++
		}
	}
	q, err := domain.BuildQuote(item, hours, inService, len(rented), code,
		domain.BookingContext{Person: p, PriorBookings: prior, Start: start})
	if err != nil {
		return q, item, p, err
	}
	return q, item, p, nil
}

func (s *MemoryStore) Quote(itemNr string, personID int, start time.Time, hours int, code string) (domain.Quote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, _, _, err := s.quoteLocked(itemNr, personID, start, hours, code)
	return q, err
}

func (s *MemoryStore) CreateRental(itemNr string, personID int, start time.Time, hours int, code string) (domain.Rental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, item, p, err := s.quoteLocked(itemNr, personID, start, hours, code)
	if err != nil {
		return domain.Rental{}, err
	}
	r := domain.Rental{
		ID: s.nextRent, ItemNr: item.Nr, ItemName: item.Name,
		PersonID: p.ID, PersonName: p.Name, Start: start, Hours: hours,
		Factor: q.Factor, Voucher: q.Voucher, DiscountPercent: q.DiscountPercent, Price: q.Total,
	}
	s.nextRent++
	s.rentals[r.ID] = r
	if q.Voucher == domain.CodeLoyalty {
		p.LoyaltyBookings = 0 // gilt als letzte Buchung: Zählung beginnt von vorn
	} else {
		p.LoyaltyBookings++
	}
	s.persons[p.ID] = p
	return r, nil
}

// ReturnRental nimmt den Artikel zurück. Mit Schadenbericht geht er außer Betrieb (zur Prüfung).
func (s *MemoryStore) ReturnRental(id int, report *domain.DamageReport) (domain.Rental, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rentals[id]
	if !ok {
		return r, ErrNotFound
	}
	if !r.Active() {
		return r, fmt.Errorf("bereits zurückgegeben: %w", ErrConflict)
	}
	now := time.Now()
	r.ReturnedAt = &now
	s.rentals[id] = r
	if report != nil {
		if it, ok := s.items[r.ItemNr]; ok {
			report.ReportedAt = now
			it.OutOfService, it.Damage = true, report
			s.items[it.Nr] = it
		}
	}
	return r, nil
}

func (s *MemoryStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := Stats{Total: len(s.items), Persons: len(s.persons)}
	now := time.Now()
	for _, it := range s.items {
		if it.OutOfService {
			st.OutOfService++
		}
	}
	st.Rented = len(s.rentedLocked())
	st.Available = st.Total - st.OutOfService - st.Rented
	for _, r := range s.rentals {
		if r.Overdue(now) {
			st.OverdueCount++
		}
		if !r.Active() {
			st.Revenue += r.Price
		}
	}
	return st
}
