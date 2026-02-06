package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/guestManager/internal/test"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var guestId = bson.NewObjectID()

func seed() map[bson.ObjectID]documents.Guest {
	return map[bson.ObjectID]documents.Guest{
		guestId: {Id: guestId, DocumentId: "1234567890", GivenNames: "John", Surname: "Smith", Email: "test@test.com"},
	}
}

func assertEqual[T any](t *testing.T, want, got T) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("assertEqual failed:\nwant: %#v\ngot:  %#v", want, got)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func assertErrorAs[T error](t *testing.T, err error) {
	t.Helper()

	var target T
	if !errors.As(err, &target) {
		var zero T
		t.Fatalf("expected error of type %T, but got: %v", zero, err)
	}
}

func setupAndRunParallel(fn func(service GuestService)) func(t *testing.T) {
	return func(t *testing.T) {
		repo := test.NewGuestRepoFake(seed())
		service := NewGuestService(repo)

		t.Parallel()
		fn(service)
	}
}

func Test_guestService_Add(t *testing.T) {
	newGuestId := bson.NewObjectID()
	newGuest, _ := domain.NewGuest(newGuestId, "0987654321", "Alice", "Doe", "alice@example.com")

	t.Run("guestService_Add adds guest", setupAndRunParallel(func(service GuestService) {
		got, err := service.Add(t.Context(), newGuest)
		assertEqual(t, got, newGuestId)
		assertNoError(t, err)
	}))

}

func Test_guestService_GetByDocument(t *testing.T) {
	guestDoc := seed()[guestId]
	guest, _ := domain.NewGuest(guestId, guestDoc.DocumentId, guestDoc.GivenNames, guestDoc.Surname, guestDoc.Email)

	testCases := map[string]struct {
		input  string
		assert func(t *testing.T, got domain.Guest, err error)
	}{
		"returns guest when document exists": {
			input: guestDoc.DocumentId,
			assert: func(t *testing.T, got domain.Guest, err error) {
				assertEqual(t, got, guest)
				assertNoError(t, err)
			},
		},
		"returns error when not found": {
			input: "does-not-exist",
			assert: func(t *testing.T, got domain.Guest, err error) {
				assertEqual(t, got, domain.Guest{})
				assertErrorAs[*dbErrors.ErrGuestDoesNotExist](t, err)
			},
		},
	}

	for caseName, tt := range testCases {
		t.Run(caseName, setupAndRunParallel(func(service GuestService) {
			got, err := service.GetByDocument(t.Context(), tt.input)
			tt.assert(t, got, err)
		}))
	}
}

func Test_guestService_GetById(t *testing.T) {
	guest := seed()[guestId]

	validGuest, _ := domain.NewGuest(guestId, guest.DocumentId, guest.GivenNames, guest.Surname, guest.Email)

	badId := bson.NewObjectID()

	tests := []struct {
		name    string
		repo    *test.GuestRepoFake
		id      bson.ObjectID
		want    domain.Guest
		wantErr bool
		errType any
	}{
		{
			name: "returns guest when id exists",
			repo: test.NewGuestRepoFake(seed()),
			id:   guestId,
			want: validGuest,
		},
		{
			name:    "returns not found error",
			repo:    test.NewGuestRepoFake(seed()),
			id:      bson.NewObjectID(),
			wantErr: true,
			errType: (*dbErrors.ErrGuestDoesNotExist)(nil),
		},
		{
			name: "returns validation error when stored seed invalid",
			repo: test.NewGuestRepoFake(map[bson.ObjectID]documents.Guest{
				badId: {Id: bson.NilObjectID, DocumentId: "invalid", GivenNames: "John", Surname: "Smith", Email: "test@test.com"},
			}),
			id:      badId,
			wantErr: true,
			errType: (*validationErrors.ErrValidationConstrain)(nil),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewGuestService(tt.repo)
			got, err := service.GetById(t.Context(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("guestService.GetById() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				switch tt.errType.(type) {
				case *dbErrors.ErrGuestDoesNotExist:
					var target *dbErrors.ErrGuestDoesNotExist
					if !errors.As(err, &target) {
						t.Fatalf("guestService.GetById() error = %v, want ErrGuestDoesNotExist", err)
					}
				case *validationErrors.ErrValidationConstrain:
					var target *validationErrors.ErrValidationConstrain
					if !errors.As(err, &target) {
						t.Fatalf("guestService.GetById() error = %v, want ErrValidationConstrain", err)
					}
				default:
					t.Fatalf("guestService.GetById() unexpected error type %T", err)
				}
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("guestService.GetById() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

//
//func Test_guestService_Update(t *testing.T) {
//	t.Parallel()
//
//	t.Run("updates guest and returns updated domain object", func(t *testing.T) {
//		repo := test.NewGuestRepoFake(seed())
//		service := NewGuestService(repo)
//		guest := seed()[guestId]
//
//		updatedGuest, err := domain.NewGuest(guestId, guest.DocumentId, "Johnny", "Doe", "johnny@test.com")
//		if err != nil {
//			t.Fatalf("domain.NewGuest() error = %v", err)
//		}
//
//		got, err := service.Update(context.Background(), guestId, updatedGuest)
//		if err != nil {
//			t.Fatalf("guestService.Update() error = %v", err)
//		}
//
//		if !reflect.DeepEqual(got, updatedGuest) {
//			t.Fatalf("guestService.Update() = %+v, want %+v", got, updatedGuest)
//		}
//
//		stored, err := repo.GetById(context.Background(), guestId)
//		if err != nil {
//			t.Fatalf("guestService.Update() repo state error = %v", err)
//		}
//
//		if !reflect.DeepEqual(stored, updatedGuest.ToMongoDoc()) {
//			t.Fatalf("guestService.Update() repo stored %+v, want %+v", stored, updatedGuest.ToMongoDoc())
//		}
//	})
//
//	t.Run("returns validation error when repository returns invalid guest", func(t *testing.T) {
//		repo := test.NewGuestRepoFake(seed())
//		service := NewGuestService(repo)
//
//		_, err := service.Update(context.Background(), guestId, domain.Guest{})
//		if err == nil {
//			t.Fatalf("guestService.Update() expected error, got nil")
//		}
//
//		var validationErr *validationErrors.ErrValidationConstrain
//		if !errors.As(err, &validationErr) {
//			t.Fatalf("guestService.Update() error = %v, want ErrValidationConstrain", err)
//		}
//	})
//}
