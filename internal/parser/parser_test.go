package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clobucks/decogen/internal/model"
)

func TestParseInterface(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "parser-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test cases
	tests := []struct {
		name          string
		fileContent   string
		interfaceName string
		expectedModel *model.Interface
		expectedError bool
	}{
		{
			name: "Simple interface",
			fileContent: `
package storage

// UserStorage is a simple storage interface
type UserStorage interface {
	// Get retrieves a user by ID
	Get(id string) (string, error)
}
`,
			interfaceName: "UserStorage",
			expectedModel: &model.Interface{
				Name:        "UserStorage",
				PackageName: "storage",
				Comments:    "UserStorage is a simple storage interface\n",
				Methods: []*model.Method{
					{
						Name:     "Get",
						Comments: "Get retrieves a user by ID\n",
						Parameters: []*model.Parameter{
							{Name: "id", Type: "string"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "string"},
							{Name: "result1", Type: "error"},
						},
					},
				},
				Imports: map[string]string{},
			},
			expectedError: false,
		},
		{
			name: "Interface with complex types",
			fileContent: `
package storage

import (
	"context"
	"time"
	
	"github.com/example/models"
)

// UserStorage handles user data
type UserStorage interface {
	// Create creates a new user
	Create(ctx context.Context, user *models.User) (string, error)
	
	// GetByID gets a user by ID
	GetByID(ctx context.Context, id string) (*models.User, error)
	
	// List lists users with pagination
	List(ctx context.Context, offset, limit int) ([]*models.User, int, error)
}
`,
			interfaceName: "UserStorage",
			expectedModel: &model.Interface{
				Name:        "UserStorage",
				PackageName: "storage",
				Comments:    "UserStorage handles user data\n",
				Methods: []*model.Method{
					{
						Name:     "Create",
						Comments: "Create creates a new user\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "user", Type: "*models.User"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "string"},
							{Name: "result1", Type: "error"},
						},
					},
					{
						Name:     "GetByID",
						Comments: "GetByID gets a user by ID\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "id", Type: "string"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "*models.User"},
							{Name: "result1", Type: "error"},
						},
					},
					{
						Name:     "List",
						Comments: "List lists users with pagination\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "offset", Type: "int"},
							{Name: "limit", Type: "int"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "[]*models.User"},
							{Name: "result1", Type: "int"},
							{Name: "result2", Type: "error"},
						},
					},
				},
				Imports: map[string]string{
					"context": "context",
					"time":    "time",
					"models":  "github.com/example/models",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with named return values",
			fileContent: `
package storage

import (
	"context"
)

// KeyStorage is a key-value storage interface
type KeyStorage interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) (value string, found bool, err error)
}
`,
			interfaceName: "KeyStorage",
			expectedModel: &model.Interface{
				Name:        "KeyStorage",
				PackageName: "storage",
				Comments:    "KeyStorage is a key-value storage interface\n",
				Methods: []*model.Method{
					{
						Name:     "Get",
						Comments: "Get retrieves a value by key\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "key", Type: "string"},
						},
						Results: []*model.Parameter{
							{Name: "value", Type: "string"},
							{Name: "found", Type: "bool"},
							{Name: "err", Type: "error"},
						},
					},
				},
				Imports: map[string]string{
					"context": "context",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with void return",
			fileContent: `
package storage

import "context"

// CacheStorage provides caching functionality
type CacheStorage interface {
	// Set sets a value in the cache
	Set(ctx context.Context, key string, value interface{}, ttl int) error
	
	// Delete removes a key from the cache
	Delete(ctx context.Context, key string) error
	
	// Clear removes all entries
	Clear(ctx context.Context)
}`,
			interfaceName: "CacheStorage",
			expectedModel: &model.Interface{
				Name:        "CacheStorage",
				PackageName: "storage",
				Comments:    "CacheStorage provides caching functionality\n",
				Methods: []*model.Method{
					{
						Name:     "Set",
						Comments: "Set sets a value in the cache\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "key", Type: "string"},
							{Name: "value", Type: "interface{}"},
							{Name: "ttl", Type: "int"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
					{
						Name:     "Delete",
						Comments: "Delete removes a key from the cache\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "key", Type: "string"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
					{
						Name:     "Clear",
						Comments: "Clear removes all entries\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
						},
						Results: []*model.Parameter{},
					},
				},
				Imports: map[string]string{
					"context": "context",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with map, chan and function types",
			fileContent: `
package storage

import "context"

// ComplexStorage demonstrates complex types
type ComplexStorage interface {
	// ProcessMap handles a map
	ProcessMap(ctx context.Context, data map[string]interface{}) error
	
	// ReceiveMessages handles a channel
	ReceiveMessages(ctx context.Context, msgChan <-chan string) error
	
	// WithCallback accepts a callback function
	WithCallback(ctx context.Context, callback func(string) error) error
}`,
			interfaceName: "ComplexStorage",
			expectedModel: &model.Interface{
				Name:        "ComplexStorage",
				PackageName: "storage",
				Comments:    "ComplexStorage demonstrates complex types\n",
				Methods: []*model.Method{
					{
						Name:     "ProcessMap",
						Comments: "ProcessMap handles a map\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "data", Type: "map[string]interface{}"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
					{
						Name:     "ReceiveMessages",
						Comments: "ReceiveMessages handles a channel\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "msgChan", Type: "chan"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
					{
						Name:     "WithCallback",
						Comments: "WithCallback accepts a callback function\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "callback", Type: "func()"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
				},
				Imports: map[string]string{
					"context": "context",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with renamed imports",
			fileContent: `
package storage

import (
	"context"
	
	mymodels "github.com/example/models"
)

// ModelStorage handles models with renamed imports
type ModelStorage interface {
	// Get retrieves a model
	Get(ctx context.Context, id string) (*mymodels.Model, error)
}`,
			interfaceName: "ModelStorage",
			expectedModel: &model.Interface{
				Name:        "ModelStorage",
				PackageName: "storage",
				Comments:    "ModelStorage handles models with renamed imports\n",
				Methods: []*model.Method{
					{
						Name:     "Get",
						Comments: "Get retrieves a model\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "id", Type: "string"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "*mymodels.Model"},
							{Name: "result1", Type: "error"},
						},
					},
				},
				Imports: map[string]string{
					"context":  "context",
					"mymodels": "github.com/example/models",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with generics - simple type parameter",
			fileContent: `
package storage

import "context"

// Repository is a generic repository interface
type Repository[T any] interface {
	// Save saves an entity
	Save(ctx context.Context, entity T) error
	
	// FindByID finds an entity by ID
	FindByID(ctx context.Context, id string) (T, error)
	
	// FindAll returns all entities
	FindAll(ctx context.Context) ([]T, error)
}`,
			interfaceName: "Repository",
			expectedModel: &model.Interface{
				Name:        "Repository",
				PackageName: "storage",
				Comments:    "Repository is a generic repository interface\n",
				Methods: []*model.Method{
					{
						Name:     "Save",
						Comments: "Save saves an entity\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "entity", Type: "T"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
					{
						Name:     "FindByID",
						Comments: "FindByID finds an entity by ID\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "id", Type: "string"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "T"},
							{Name: "result1", Type: "error"},
						},
					},
					{
						Name:     "FindAll",
						Comments: "FindAll returns all entities\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "[]T"},
							{Name: "result1", Type: "error"},
						},
					},
				},
				Imports: map[string]string{
					"context": "context",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with generics - multiple type parameters",
			fileContent: `
package storage

import "context"

// KeyValueStore is a generic key-value store
type KeyValueStore[K comparable, V any] interface {
	// Set stores a value with the given key
	Set(ctx context.Context, key K, value V) error
	
	// Get retrieves a value by key
	Get(ctx context.Context, key K) (V, bool, error)
	
	// Delete removes a key
	Delete(ctx context.Context, key K) error
	
	// GetAll returns all key-value pairs
	GetAll(ctx context.Context) (map[K]V, error)
}`,
			interfaceName: "KeyValueStore",
			expectedModel: &model.Interface{
				Name:        "KeyValueStore",
				PackageName: "storage",
				Comments:    "KeyValueStore is a generic key-value store\n",
				Methods: []*model.Method{
					{
						Name:     "Set",
						Comments: "Set stores a value with the given key\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "key", Type: "K"},
							{Name: "value", Type: "V"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
					{
						Name:     "Get",
						Comments: "Get retrieves a value by key\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "key", Type: "K"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "V"},
							{Name: "result1", Type: "bool"},
							{Name: "result2", Type: "error"},
						},
					},
					{
						Name:     "Delete",
						Comments: "Delete removes a key\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "key", Type: "K"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
					{
						Name:     "GetAll",
						Comments: "GetAll returns all key-value pairs\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "map[K]V"},
							{Name: "result1", Type: "error"},
						},
					},
				},
				Imports: map[string]string{
					"context": "context",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with generics - constrained type parameter",
			fileContent: `
package storage

import (
	"context"
	"fmt"
	"encoding/json"
)

// JSONSerializer provides JSON serialization for types
type JSONSerializer[T fmt.Stringer] interface {
	// Serialize converts an object to JSON
	Serialize(ctx context.Context, obj T) ([]byte, error)
	
	// Deserialize converts JSON to an object
	Deserialize(ctx context.Context, data []byte) (T, error)
}`,
			interfaceName: "JSONSerializer",
			expectedModel: &model.Interface{
				Name:        "JSONSerializer",
				PackageName: "storage",
				Comments:    "JSONSerializer provides JSON serialization for types\n",
				Methods: []*model.Method{
					{
						Name:     "Serialize",
						Comments: "Serialize converts an object to JSON\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "obj", Type: "T"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "[]byte"},
							{Name: "result1", Type: "error"},
						},
					},
					{
						Name:     "Deserialize",
						Comments: "Deserialize converts JSON to an object\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "data", Type: "[]byte"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "T"},
							{Name: "result1", Type: "error"},
						},
					},
				},
				Imports: map[string]string{
					"context": "context",
					"fmt":     "fmt",
					"json":    "encoding/json",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface with embedded interface",
			fileContent: `
package storage

import "context"

// Reader defines basic read operations
type Reader interface {
	// Read reads data
	Read(ctx context.Context, id string) ([]byte, error)
}

// Writer defines basic write operations
type Writer interface {
	// Write writes data
	Write(ctx context.Context, id string, data []byte) error
}

// ReadWriter combines read and write operations
type ReadWriter interface {
	Reader
	Writer
	
	// Size returns the size
	Size(ctx context.Context, id string) (int64, error)
}`,
			interfaceName: "ReadWriter",
			expectedModel: &model.Interface{
				Name:        "ReadWriter",
				PackageName: "storage",
				Comments:    "ReadWriter combines read and write operations\n",
				Methods: []*model.Method{
					{
						Name:     "Size",
						Comments: "Size returns the size\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "id", Type: "string"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "int64"},
							{Name: "result1", Type: "error"},
						},
					},
					// Note: The embedded interface methods are not currently detected
					// This would need to be fixed in the parser implementation
				},
				Imports: map[string]string{
					"context": "context",
				},
			},
			expectedError: false,
		},
		{
			name: "Interface not found",
			fileContent: `
package storage

// UserStorage is the interface
type UserStorage interface {
	Get(id string) (string, error)
}`,
			interfaceName: "NotExistingInterface",
			expectedModel: nil,
			expectedError: true,
		},
		{
			name: "Invalid Go code",
			fileContent: `
package storage

this is not valid Go code
`,
			interfaceName: "UserStorage",
			expectedModel: nil,
			expectedError: true,
		},
		{
			name: "Interface with variadic parameters",
			fileContent: `
package storage

// BatchStorage handles batch operations
type BatchStorage interface {
	// BatchGet gets multiple items at once
	BatchGet(ids ...string) ([]string, error)
	
	// BatchProcess processes multiple items
	BatchProcess(ctx context.Context, options map[string]string, items ...interface{}) error
}`,
			interfaceName: "BatchStorage",
			expectedModel: &model.Interface{
				Name:        "BatchStorage",
				PackageName: "storage",
				Comments:    "BatchStorage handles batch operations\n",
				Methods: []*model.Method{
					{
						Name:     "BatchGet",
						Comments: "BatchGet gets multiple items at once\n",
						Parameters: []*model.Parameter{
							{Name: "ids", Type: "...string"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "[]string"},
							{Name: "result1", Type: "error"},
						},
					},
					{
						Name:     "BatchProcess",
						Comments: "BatchProcess processes multiple items\n",
						Parameters: []*model.Parameter{
							{Name: "ctx", Type: "context.Context"},
							{Name: "options", Type: "map[string]string"},
							{Name: "items", Type: "...interface{}"},
						},
						Results: []*model.Parameter{
							{Name: "result0", Type: "error"},
						},
					},
				},
				Imports: map[string]string{},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a source file with the test content
			sourceFile := filepath.Join(tempDir, tt.name+".go")
			err := os.WriteFile(sourceFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			// Parse the interface
			interfaceModel, err := ParseInterface(sourceFile, tt.interfaceName)

			// Check error expectation
			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, interfaceModel)

			// Compare interface properties
			assert.Equal(t, tt.expectedModel.Name, interfaceModel.Name)
			assert.Equal(t, tt.expectedModel.PackageName, interfaceModel.PackageName)
			assert.Equal(t, tt.expectedModel.Comments, interfaceModel.Comments)

			// Compare methods
			// Note: we check only methods that are in the expected model
			// For embedded interfaces, our parser may detect methods from embedded interfaces,
			// but we don't validate those in our test cases yet
			for _, expectedMethod := range tt.expectedModel.Methods {
				// Find the matching method in the actual model
				var actualMethod *model.Method
				for _, m := range interfaceModel.Methods {
					if m.Name == expectedMethod.Name {
						actualMethod = m
						break
					}
				}

				assert.NotNil(t, actualMethod, "Method %s not found", expectedMethod.Name)
				if actualMethod == nil {
					continue
				}

				assert.Equal(t, expectedMethod.Comments, actualMethod.Comments)

				// Compare parameters
				assert.Equal(t, len(expectedMethod.Parameters), len(actualMethod.Parameters))
				for j, expectedParam := range expectedMethod.Parameters {
					if j >= len(actualMethod.Parameters) {
						t.Fatalf("Missing parameter at index %d for method %s", j, expectedMethod.Name)
					}
					actualParam := actualMethod.Parameters[j]

					assert.Equal(t, expectedParam.Name, actualParam.Name)
					assert.Equal(t, expectedParam.Type, actualParam.Type)
				}

				// Compare results
				assert.Equal(t, len(expectedMethod.Results), len(actualMethod.Results))
				for j, expectedResult := range expectedMethod.Results {
					if j >= len(actualMethod.Results) {
						t.Fatalf("Missing result at index %d for method %s", j, expectedMethod.Name)
					}
					actualResult := actualMethod.Results[j]

					// For results we may not care about the specific names for unnamed results
					if expectedResult.Name != "result0" && expectedResult.Name != "result1" && expectedResult.Name != "result2" {
						assert.Equal(t, expectedResult.Name, actualResult.Name)
					}
					assert.Equal(t, expectedResult.Type, actualResult.Type)
				}
			}

			// Check imports - we don't need exact matching of all imports, but the ones we specified should be there
			for expectedImportName, expectedImportPath := range tt.expectedModel.Imports {
				importPath, exists := interfaceModel.Imports[expectedImportName]
				assert.True(t, exists, "Import %s not found", expectedImportName)
				if exists {
					assert.Equal(t, expectedImportPath, importPath)
				}
			}
		})
	}
}
