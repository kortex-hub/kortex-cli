---
name: testing-best-practices
description: Testing best practices including parallel execution, fake objects, and factory injection patterns
argument-hint: ""
---

# Testing Best Practices

This skill covers general testing best practices for the kdn project, including parallel test execution and using fake objects for testability.

## Parallel Test Execution

**All tests MUST call `t.Parallel()` as the first line of the test function.**

This ensures faster test execution and better resource utilization. Every test function should start with:

```go
func TestExample(t *testing.T) {
    t.Parallel()

    // Test code here...
}
```

### Benefits of Parallel Tests

- **Faster CI/CD**: Tests run concurrently, reducing total execution time
- **Better resource utilization**: Makes use of multi-core systems
- **Identifies race conditions**: Helps find concurrency issues early

### Exception: Tests Using `t.Setenv()`

Tests that use `t.Setenv()` to set environment variables **cannot use `t.Parallel()`** on the parent test function. The Go testing framework enforces this restriction because environment variable changes affect the entire process.

```go
// CORRECT: No t.Parallel() when using t.Setenv()
func TestWithEnvVariable(t *testing.T) {
    t.Run("subtest with env var", func(t *testing.T) {
        t.Setenv("MY_VAR", "value")
        // Test code here...
    })
}

// INCORRECT: Will panic at runtime
func TestWithEnvVariable(t *testing.T) {
    t.Parallel() // ❌ WRONG - cannot use with t.Setenv()

    t.Run("subtest with env var", func(t *testing.T) {
        t.Setenv("MY_VAR", "value")
        // Test code here...
    })
}
```

**Reference:** See `pkg/cmd/root_test.go:TestRootCmd_StorageEnvVariable()` for an example of testing with environment variables.

## Testing with Fake Objects

When testing code that uses interfaces (following the Module Design Pattern), **use fake implementations instead of real implementations or mocks**.

### Why Fakes Over Mocks

- **No external dependencies** - Fakes are simple structs with no framework requirements
- **Full control** - Control exact behavior through fields/parameters
- **Type-safe** - Compile-time verification that fakes implement interfaces
- **Easy to understand** - Fakes are just plain Go code
- **Flexible** - Can create different factories for different test scenarios

### Pattern

1. Create unexported fake structs that implement the interface
2. Use factory injection to provide fakes to the code under test
3. Control fake behavior through constructor parameters or fields

### Example: Fake Instance Factory

```go
// Fake instance factory for testing
type fakeInstanceFactory struct {
    instances map[string]instances.Instance
    err       error
}

func newFakeInstanceFactory() *fakeInstanceFactory {
    return &fakeInstanceFactory{
        instances: make(map[string]instances.Instance),
    }
}

func (f *fakeInstanceFactory) NewInstance(params instances.NewInstanceParams) (instances.Instance, error) {
    if f.err != nil {
        return nil, f.err
    }

    instance := &fakeInstance{
        id:        generateID(params.SourceDir),
        sourceDir: params.SourceDir,
        configDir: params.ConfigDir,
    }

    f.instances[instance.id] = instance
    return instance, nil
}

// Use in tests
func TestManager_Add(t *testing.T) {
    t.Parallel()

    storageDir := t.TempDir()
    fakeFactory := newFakeInstanceFactory()

    manager, err := newManagerWithFactory(storageDir, fakeFactory)
    if err != nil {
        t.Fatalf("Failed to create manager: %v", err)
    }

    // Test using the fake factory
    instance, err := manager.Add(context.Background(), instances.AddOptions{
        // ...
    })
}
```

### Example: Fake with Controlled Errors

```go
// Fake that can be configured to return errors
type fakeGenerator struct {
    nextID string
    err    error
}

func newFakeGenerator() *fakeGenerator {
    return &fakeGenerator{
        nextID: "test-id",
    }
}

func newFakeGeneratorWithError(err error) *fakeGenerator {
    return &fakeGenerator{
        err: err,
    }
}

func (f *fakeGenerator) Generate(input string) (string, error) {
    if f.err != nil {
        return "", f.err
    }
    return f.nextID, nil
}

// Use in tests
func TestManager_Add_GeneratorError(t *testing.T) {
    t.Parallel()

    storageDir := t.TempDir()
    fakeGen := newFakeGeneratorWithError(errors.New("generator error"))

    manager, err := newManagerWithFactory(storageDir, fakeInstanceFactory, fakeGen)
    if err != nil {
        t.Fatalf("Failed to create manager: %v", err)
    }

    _, err = manager.Add(context.Background(), instances.AddOptions{
        // ...
    })

    if err == nil {
        t.Fatal("Expected error from generator")
    }
}
```

### Example: Fake with Behavior Tracking

```go
// Fake that tracks method calls
type fakeRuntime struct {
    createCalls []runtime.CreateParams
    startCalls  []string
    stopCalls   []string
}

func newFakeRuntime() *fakeRuntime {
    return &fakeRuntime{
        createCalls: []runtime.CreateParams{},
        startCalls:  []string{},
        stopCalls:   []string{},
    }
}

func (f *fakeRuntime) Create(ctx context.Context, params runtime.CreateParams) (runtime.RuntimeInfo, error) {
    f.createCalls = append(f.createCalls, params)
    return runtime.RuntimeInfo{
        ID:    "test-id",
        State: "created",
    }, nil
}

func (f *fakeRuntime) Start(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
    f.startCalls = append(f.startCalls, id)
    return runtime.RuntimeInfo{
        ID:    id,
        State: "running",
    }, nil
}

// Use in tests
func TestManager_Lifecycle(t *testing.T) {
    t.Parallel()

    fakeRT := newFakeRuntime()

    // ... test code that uses the runtime ...

    // Verify behavior
    if len(fakeRT.createCalls) != 1 {
        t.Errorf("Expected 1 Create() call, got %d", len(fakeRT.createCalls))
    }

    if len(fakeRT.startCalls) != 1 {
        t.Errorf("Expected 1 Start() call, got %d", len(fakeRT.startCalls))
    }
}
```

### Factory Injection Pattern

Use factory injection to provide fakes to the code under test:

```go
// Production constructor
func NewManager(storageDir string) (Manager, error) {
    return newManagerWithFactory(
        storageDir,
        instances.NewInstance,  // Real factory
        generator.New(),        // Real generator
        registry.New(),         // Real registry
        git.NewDetector(),      // Real detector
    )
}

// Test-friendly constructor with factory injection
func newManagerWithFactory(
    storageDir string,
    instanceFactory instanceFactory,
    generator idGenerator,
    registry runtimeRegistry,
    detector git.Detector,
) (Manager, error) {
    // Implementation uses injected dependencies
}

// In tests
func TestManager(t *testing.T) {
    t.Parallel()

    fakeFactory := newFakeInstanceFactory()
    fakeGen := newFakeGenerator()
    fakeRegistry := newTestRegistry(t.TempDir())
    fakeDetector := newFakeGitDetector()

    manager, err := newManagerWithFactory(
        t.TempDir(),
        fakeFactory,
        fakeGen,
        fakeRegistry,
        fakeDetector,
    )
    // ... test code ...
}
```

## Test Organization

### Naming Conventions

```go
// Unit tests for specific methods
func TestManager_Add(t *testing.T) { ... }
func TestManager_Delete(t *testing.T) { ... }

// E2E tests
func TestInitCmd_E2E(t *testing.T) { ... }

// PreRun validation tests
func TestInitCmd_PreRun(t *testing.T) { ... }

// Example validation tests
func TestInitCmd_Examples(t *testing.T) { ... }
```

### Test Structure

```go
func TestManager_Add(t *testing.T) {
    t.Parallel()

    t.Run("adds instance successfully", func(t *testing.T) {
        t.Parallel()

        // Arrange
        storageDir := t.TempDir()
        manager, _ := NewManager(storageDir)

        // Act
        instance, err := manager.Add(context.Background(), options)

        // Assert
        if err != nil {
            t.Fatalf("Add() failed: %v", err)
        }
        if instance.ID == "" {
            t.Error("Expected non-empty ID")
        }
    })

    t.Run("returns error for invalid input", func(t *testing.T) {
        t.Parallel()

        // Arrange
        storageDir := t.TempDir()
        manager, _ := NewManager(storageDir)

        // Act
        _, err := manager.Add(context.Background(), invalidOptions)

        // Assert
        if err == nil {
            t.Fatal("Expected error for invalid input")
        }
    })
}
```

## Table-Driven Tests

Use table-driven tests for testing multiple scenarios:

```go
func TestValidate(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        input   string
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid input",
            input:   "valid",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
            errMsg:  "cannot be empty",
        },
        {
            name:    "invalid characters",
            input:   "invalid@#$",
            wantErr: true,
            errMsg:  "invalid characters",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            err := Validate(tt.input)

            if (err != nil) != tt.wantErr {
                t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }

            if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
                t.Errorf("Expected error to contain '%s', got: %s", tt.errMsg, err.Error())
            }
        })
    }
}
```

## Test Helpers

Create test helpers to reduce boilerplate:

```go
// Test helper for creating a test manager
func newTestManager(t *testing.T) instances.Manager {
    t.Helper()

    storageDir := t.TempDir()
    manager, err := instances.NewManager(storageDir)
    if err != nil {
        t.Fatalf("Failed to create manager: %v", err)
    }

    return manager
}

// Test helper for adding a test instance
func addTestInstance(t *testing.T, manager instances.Manager) instances.Instance {
    t.Helper()

    sourceDir := t.TempDir()
    configDir := t.TempDir()

    instance, err := instances.NewInstance(instances.NewInstanceParams{
        SourceDir: sourceDir,
        ConfigDir: configDir,
    })
    if err != nil {
        t.Fatalf("Failed to create instance: %v", err)
    }

    added, err := manager.Add(context.Background(), instances.AddOptions{
        Instance:    instance,
        RuntimeType: "fake",
    })
    if err != nil {
        t.Fatalf("Failed to add instance: %v", err)
    }

    return added
}

// Use in tests
func TestManager_Delete(t *testing.T) {
    t.Parallel()

    manager := newTestManager(t)
    instance := addTestInstance(t, manager)

    err := manager.Delete(instance.ID)
    if err != nil {
        t.Fatalf("Delete() failed: %v", err)
    }
}
```

## Cleanup and Resource Management

Use `t.TempDir()` for automatic cleanup:

```go
func TestWithFiles(t *testing.T) {
    t.Parallel()

    // Automatically cleaned up after test
    tempDir := t.TempDir()

    // Create test files
    testFile := filepath.Join(tempDir, "test.txt")
    os.WriteFile(testFile, []byte("test"), 0644)

    // No need to manually clean up - t.TempDir() handles it
}
```

Use `t.Cleanup()` for custom cleanup:

```go
func TestWithCleanup(t *testing.T) {
    t.Parallel()

    // Custom cleanup
    t.Cleanup(func() {
        // Clean up resources
    })

    // Test code...
}
```

## Related Skills

- `/testing-commands` - Command-specific testing patterns
- `/cross-platform-development` - Cross-platform testing practices

## References

- **Parallel Tests**: All `*_test.go` files should use `t.Parallel()`
- **Fake Objects**: `pkg/instances/manager_test.go` for complete examples
- **Factory Injection**: `pkg/instances/manager.go` and `pkg/instances/manager_test.go`
- **Go Testing Package**: Standard library documentation
