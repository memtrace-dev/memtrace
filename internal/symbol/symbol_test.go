package symbol

import (
	"testing"
)

// --- Go extractor ---

func TestExtractGo_Functions(t *testing.T) {
	src := []byte(`package main

// HandleAuth handles authentication requests.
func HandleAuth(w http.ResponseWriter, r *http.Request) {}

func helper() {} // unexported, still extracted

func (s *Server) Start() error { return nil }
`)
	syms, err := extractGo("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := symbolNames(syms)
	assertContains(t, names, "HandleAuth")
	assertContains(t, names, "helper")
	assertContains(t, names, "Start")
}

func TestExtractGo_DocComment(t *testing.T) {
	src := []byte(`package main

// ValidateToken checks the JWT token signature.
func ValidateToken(token string) bool { return true }
`)
	syms, err := extractGo("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(syms) == 0 {
		t.Fatal("expected symbols")
	}
	if syms[0].Comment == "" {
		t.Errorf("expected doc comment, got empty")
	}
}

func TestExtractGo_Types(t *testing.T) {
	src := []byte(`package main

type Server struct { port int }
type Handler interface { ServeHTTP() }
type Config = map[string]string
`)
	syms, err := extractGo("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kinds := symbolKinds(syms)
	assertContainsKind(t, kinds, KindStruct)
	assertContainsKind(t, kinds, KindInterface)
}

func TestExtractGo_ExportedConsts(t *testing.T) {
	src := []byte(`package main

const (
	MaxRetries = 3
	timeout    = 10 // unexported — skipped
)
`)
	syms, err := extractGo("test.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := symbolNames(syms)
	assertContains(t, names, "MaxRetries")
	for _, n := range names {
		if n == "timeout" {
			t.Errorf("unexported const 'timeout' should not be extracted")
		}
	}
}

// --- TypeScript extractor ---

func TestExtractTypeScript_Function(t *testing.T) {
	src := []byte(`
export function validateToken(token: string): boolean {
  return true
}

export async function fetchUser(id: string): Promise<User> {}
`)
	syms, err := extractTypeScript(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := symbolNames(syms)
	assertContains(t, names, "validateToken")
	assertContains(t, names, "fetchUser")
}

func TestExtractTypeScript_Class(t *testing.T) {
	src := []byte(`
export class AuthService {
  constructor(private db: DB) {}
}

export abstract class BaseController {}
`)
	syms, err := extractTypeScript(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := symbolNames(syms)
	assertContains(t, names, "AuthService")
	assertContains(t, names, "BaseController")
}

func TestExtractTypeScript_InterfaceAndType(t *testing.T) {
	src := []byte(`
export interface User {
  id: string
  name: string
}

export type AuthToken = string
export type Config = { key: string }
`)
	syms, err := extractTypeScript(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := symbolNames(syms)
	assertContains(t, names, "User")
	assertContains(t, names, "AuthToken")
}

// --- Python extractor ---

func TestExtractPython_Functions(t *testing.T) {
	src := []byte(`
def validate_token(token):
    return True

async def fetch_user(user_id):
    pass

def _private():  # should be skipped
    pass

class AuthService:
    pass
`)
	syms, err := extractPython(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := symbolNames(syms)
	assertContains(t, names, "validate_token")
	assertContains(t, names, "fetch_user")
	assertContains(t, names, "AuthService")
	for _, n := range names {
		if n == "_private" {
			t.Errorf("private function '_private' should be skipped")
		}
	}
}

// --- Rust extractor ---

func TestExtractRust_FunctionsAndTypes(t *testing.T) {
	src := []byte(`
pub fn validate_token(token: &str) -> bool { true }

pub async fn fetch_user(id: u64) -> Option<User> { None }

pub struct AuthService {
    db: Database,
}

pub enum TokenKind { Bearer, Basic }

pub trait Validator {
    fn validate(&self) -> bool;
}
`)
	syms, err := extractRust(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := symbolNames(syms)
	assertContains(t, names, "validate_token")
	assertContains(t, names, "fetch_user")
	assertContains(t, names, "AuthService")
	assertContains(t, names, "TokenKind")
	assertContains(t, names, "Validator")
}

// --- dispatch via Extract ---

func TestExtract_UnsupportedReturnsNil(t *testing.T) {
	syms, err := Extract("README.md", []byte("# hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if syms != nil {
		t.Errorf("expected nil for unsupported file type, got %v", syms)
	}
}

func TestExtract_GoDispatch(t *testing.T) {
	src := []byte("package main\nfunc Foo() {}")
	syms, err := Extract("main.go", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(syms) == 0 {
		t.Error("expected at least one symbol")
	}
}

// --- MemoryContent and Tags ---

func TestMemoryContent_WithComment(t *testing.T) {
	sym := Symbol{Name: "ValidateJWT", Kind: KindFunction, Comment: "ValidateJWT checks the token."}
	got := MemoryContent(sym, "src/auth/middleware.go")
	if got == "" {
		t.Error("expected non-empty content")
	}
	if got[0:8] != "Function" {
		t.Errorf("expected to start with 'Function', got: %s", got)
	}
}

func TestTags_Go(t *testing.T) {
	sym := Symbol{Name: "Foo", Kind: KindFunction}
	tags := Tags(sym, "src/foo.go")
	assertContains(t, tags, "symbol")
	assertContains(t, tags, "function")
	assertContains(t, tags, "go")
}

// --- helpers ---

func symbolNames(syms []Symbol) []string {
	names := make([]string, len(syms))
	for i, s := range syms {
		names[i] = s.Name
	}
	return names
}

func symbolKinds(syms []Symbol) []Kind {
	kinds := make([]Kind, len(syms))
	for i, s := range syms {
		kinds[i] = s.Kind
	}
	return kinds
}

func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			return
		}
	}
	t.Errorf("expected %q in %v", needle, haystack)
}

func assertContainsKind(t *testing.T, kinds []Kind, want Kind) {
	t.Helper()
	for _, k := range kinds {
		if k == want {
			return
		}
	}
	t.Errorf("expected kind %q in %v", want, kinds)
}
