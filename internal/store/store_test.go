package store

import (
	"errors"
	"testing"
	"time"

	"verleih/internal/domain"
)

func setup(t *testing.T, n int) *MemoryStore {
	s := NewMemoryStore()
	for i := 1; i <= n; i++ {
		it := domain.Item{Nr: "A-" + string(rune('0'+i)), Name: "Test", Type: domain.AppConfig.Types[0], PriceModel: domain.PerHour, Rate: 10}
		if err := s.AddItem(it); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.AddPerson(domain.Person{Name: "Max", Birthday: "1990-01-01", IDType: "Reisepass", IDNumber: "42"}); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRentalFlow(t *testing.T) {
	s := setup(t, 4)
	r, err := s.CreateRental("A-1", 1, time.Now(), 3, "")
	if err != nil || r.Price != 30 {
		t.Fatalf("Ausleihe fehlgeschlagen: %v %v", r, err)
	}
	if _, err := s.CreateRental("A-1", 1, time.Now(), 1, ""); !errors.Is(err, ErrConflict) {
		t.Error("Doppelausleihe hätte Konflikt geben müssen")
	}
	if _, err := s.CreateRental("A-2", 1, time.Now(), 1, ""); err != nil {
		t.Error("eine Person darf mehrere Artikel gleichzeitig buchen")
	}
	if st := s.Stats(); st.Rented != 2 || st.Available != 2 {
		t.Errorf("Stats falsch: %+v", st)
	}
}

func TestDynamicPriceAtBooking(t *testing.T) {
	s := setup(t, 4)
	_, _ = s.CreateRental("A-1", 1, time.Now(), 1, "")
	_, _ = s.CreateRental("A-2", 1, time.Now(), 1, "")
	r, _ := s.CreateRental("A-3", 1, time.Now(), 1, "") // 2 von 4 vermietet = 50 % -> noch Faktor 1.0
	if r.Price != 10 {
		t.Errorf("bei 50 %% noch Basispreis, got %v", r.Price)
	}
	r, _ = s.CreateRental("A-4", 1, time.Now(), 1, "") // 3 von 4 -> nur noch einer frei -> 150 %
	if r.Price != 15 {
		t.Errorf("letzter freier Artikel kostet 150 %%, got %v", r.Price)
	}
}

func TestLoyaltyEveryThirdBooking(t *testing.T) {
	s := setup(t, 1)
	book := func(code string) (domain.Rental, error) {
		r, err := s.CreateRental("A-1", 1, time.Now(), 1, code)
		if err == nil {
			_, _ = s.ReturnRental(r.ID, nil)
		}
		return r, err
	}
	if _, err := book(domain.CodeLoyalty); err == nil {
		t.Error("LOYALTY50 darf bei der ersten Buchung nicht gelten")
	}
	_, _ = book("")
	_, _ = book("")
	r, err := book(domain.CodeLoyalty)
	if err != nil || r.DiscountPercent != 50 {
		t.Fatalf("dritte Buchung muss 50 %% bekommen: %+v %v", r, err)
	}
	if _, err := book(domain.CodeLoyalty); err == nil {
		t.Error("direkt danach darf LOYALTY50 nicht noch einmal gelten")
	}
}

func TestDamageReportAndReview(t *testing.T) {
	s := setup(t, 2)
	r, _ := s.CreateRental("A-1", 1, time.Now(), 1, "")
	if _, err := s.ReturnRental(r.ID, &domain.DamageReport{}); err != nil { // Bericht ohne Text/Foto zählt
		t.Fatal(err)
	}
	if _, err := s.CreateRental("A-1", 1, time.Now(), 1, ""); !errors.Is(err, ErrConflict) {
		t.Error("Artikel außer Betrieb darf nicht gebucht werden")
	}
	if st := s.Stats(); st.OutOfService != 1 || st.Available != 1 {
		t.Errorf("Stats falsch: %+v", st)
	}
	if err := s.ReviewDone("A-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateRental("A-1", 1, time.Now(), 1, ""); err != nil {
		t.Error("nach der Prüfung wieder buchbar")
	}
}

func TestDeleteRentedItem(t *testing.T) {
	s := setup(t, 1)
	_, _ = s.CreateRental("A-1", 1, time.Now(), 1, "")
	if err := s.DeleteItem("A-1"); !errors.Is(err, ErrConflict) {
		t.Error("Löschen eines ausgeliehenen Artikels muss scheitern")
	}
	if err := s.DeletePerson(1); !errors.Is(err, ErrConflict) {
		t.Error("Löschen einer Person mit aktiver Ausleihe muss scheitern")
	}
}
