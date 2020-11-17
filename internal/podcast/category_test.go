package podcast

import (
	"reflect"
	"testing"

	"github.com/sschwartz96/syncapod-backend/internal/db"
)

var catCon = newCategoryCache(
	[]db.Category{
		{ID: 0, ParentID: 0, Name: "nil"},
		{ID: 1, ParentID: 0, Name: "News"},
		{ID: 2, ParentID: 1, Name: "Tech News"},
		{ID: 3, ParentID: 1, Name: "Sports News"},
		{ID: 4, ParentID: 0, Name: "True Crime"},
		{ID: 5, ParentID: 0, Name: "Sports"},
		{ID: 6, ParentID: 5, Name: "Baseball"},
	},
)

func TestCategoryController_LookupIDs(t *testing.T) {
	tests := []struct {
		name    string
		ids     []int
		want    []Category
		wantErr bool
	}{
		{
			name: "valid/all",
			ids:  []int{1, 2, 3, 4, 5, 6},
			want: []Category{
				{
					ID:   1,
					Name: "News",
					Subcategories: []Category{
						{2, "Tech News", nil},
						{3, "Sports News", nil},
					},
				},
				{
					ID:            4,
					Name:          "True Crime",
					Subcategories: []Category{},
				},
				{
					ID:   5,
					Name: "Sports",
					Subcategories: []Category{
						{6, "Baseball", nil},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid/news",
			ids:  []int{1, 2, 3},
			want: []Category{
				{
					ID:   1,
					Name: "News",
					Subcategories: []Category{
						{2, "Tech News", nil},
						{3, "Sports News", nil},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := catCon.LookupIDs(tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("CategoryController.LookupIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CategoryController.LookupIDs() = \n%v, \nwant \n%v", got, tt.want)
			}
		})
	}
}

func TestCategoryController_TranslateCategories(t *testing.T) {
	tests := []struct {
		name    string
		cats    []Category
		ids     []int
		want    []int
		wantErr bool
	}{
		{
			name: "valid/news",
			cats: []Category{
				{
					ID:   1,
					Name: "News",
					Subcategories: []Category{
						{ID: 2, Name: "Tech News", Subcategories: nil},
						{ID: 3, Name: "Sports News", Subcategories: nil},
					},
				},
			},
			ids:  []int{},
			want: []int{1, 2, 3},
		},
		{
			name: "valid/all",
			cats: []Category{
				{
					ID:   1,
					Name: "News",
					Subcategories: []Category{
						{ID: 2, Name: "Tech News", Subcategories: nil},
						{ID: 3, Name: "Sports News", Subcategories: nil},
					},
				},
				{
					ID:            4,
					Name:          "True Crime",
					Subcategories: []Category{},
				},
				{
					ID:   5,
					Name: "Sports",
					Subcategories: []Category{
						{ID: 6, Name: "Baseball", Subcategories: nil},
					},
				},
			},
			ids:  []int{},
			want: []int{1, 2, 3, 4, 5, 6},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := catCon.TranslateCategories(tt.cats, tt.ids)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CategoryController.TranslateCategories() = %v, want %v", got, tt.want)
			}
		})
	}
}
