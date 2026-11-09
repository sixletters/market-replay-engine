package cli

import (
	"errors"
	"testing"

	"sixletters/back-to-the-past/internal/model"
)

func TestCheckInvariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		view    model.BookView
		wantErr bool
	}{
		{
			name: "valid book",
			view: model.BookView{
				Bids: []model.PriceLevel{{Price: 100, Quantity: 5}, {Price: 99, Quantity: 10}},
				Asks: []model.PriceLevel{{Price: 102, Quantity: 4}, {Price: 103, Quantity: 6}},
			},
		},
		{
			name: "empty book",
			view: model.BookView{},
		},
		{
			name: "bids not descending",
			view: model.BookView{
				Bids: []model.PriceLevel{{Price: 99, Quantity: 5}, {Price: 100, Quantity: 10}},
				Asks: []model.PriceLevel{{Price: 102, Quantity: 4}},
			},
			wantErr: true,
		},
		{
			name: "asks not ascending",
			view: model.BookView{
				Bids: []model.PriceLevel{{Price: 100, Quantity: 5}},
				Asks: []model.PriceLevel{{Price: 103, Quantity: 4}, {Price: 102, Quantity: 6}},
			},
			wantErr: true,
		},
		{
			name: "zero quantity bid",
			view: model.BookView{
				Bids: []model.PriceLevel{{Price: 100, Quantity: 0}},
			},
			wantErr: true,
		},
		{
			name: "crossed book",
			view: model.BookView{
				Bids: []model.PriceLevel{{Price: 105, Quantity: 5}},
				Asks: []model.PriceLevel{{Price: 102, Quantity: 4}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkInvariants(tt.view)
			if tt.wantErr && !errors.Is(err, errInvariant) {
				t.Fatalf("checkInvariants() = %v, want errInvariant", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("checkInvariants() = %v, want nil", err)
			}
		})
	}
}
