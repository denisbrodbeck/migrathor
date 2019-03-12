package migrathor

import (
	"reflect"
	"testing"
)

func Test_filterExcept(t *testing.T) {
	tests := []struct {
		name   string
		items  []string
		except []string
		want   []string
	}{
		{
			name:   "empty",
			items:  []string{},
			except: []string{},
			want:   []string{},
		},
		{
			name:   "only in items",
			items:  []string{"1", "2", "3"},
			except: []string{},
			want:   []string{"1", "2", "3"},
		},
		{
			name:   "only in except",
			items:  []string{},
			except: []string{"1", "2", "3"},
			want:   []string{},
		},
		{
			name:   "normal",
			items:  []string{"1", "2", "3", "4"},
			except: []string{"1", "3"},
			want:   []string{"2", "4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterExcept(tt.items, tt.except); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterExcept()\ngot  %v\nwant %v\n", got, tt.want)
			}
		})
	}
}
