package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	_ "github.com/godror/godror"
	"github.com/klauspost/cpuid/v2"

	_ "time/tzdata"
)

func main() {
	// set TZ env var
	// TZ not set, system /etc/localtime present System's local timezone (e.g., container's default)
	// TZ='' (empty string) UTC
	loc, err := time.LoadLocation(`UTC`)
	if err != nil {
		panic(err)
	}
	time.Local = loc // sets the global timezone to UTC to no need for Dockerfile: ENV TZ=UTC
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo})))
	slog.Info(`Go`, `Version`, runtime.Version(), `OS`, runtime.GOOS, `ARCH`, runtime.GOARCH, `GOAMD64`, AMD64Level(), `now`, time.Now(), `Local`, time.Local)

	s := os.Getenv(`DB_TIMEOUT`)
	timeout, err := time.ParseDuration(s)
	if err != nil {
		slog.Error(`timeout_env_er`, `DB_TIMEOUT`, s, `error`, err)
		return
	}

	s = os.Getenv(`DEMO_ORACLE_USER`)
	q := s
	if isOracleUserNameNeedsQuoting(s) {
		q := doubleQuoted(s)
		if s != q {
			os.Setenv(`DEMO_ORACLE_USER`, q)
		}
	}

	dsn := os.ExpandEnv(`$DEMO_ORACLE_USER/$DEMO_ORACLE_PASSWORD@$DEMO_ORACLE_SERVER/$DEMO_ORACLE_SERVICE_NAME`)
	fmt.Println(dsn)
	db, err := sql.Open("godror", dsn)
	if err != nil {
		slog.Error(`db_open_er`, `error`, err)
		return
	}

	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		slog.Error(`db_ping_er`, `error`, err)
		return
	}
	slog.Info(`ping_ok`)

	rows, err := db.Query("SELECT banner FROM V$VERSION")
	if err != nil {
		slog.Error(`db_query_version_er`, `error`, err)
		return
	}
	defer rows.Close()

	fmt.Println("\nOracle Database Version Information:")
	for rows.Next() {
		var banner string
		err := rows.Scan(&banner)
		if err != nil {
			slog.Error(`db_scan_er`, `error`, err)
			return
		}
		fmt.Println(banner)
	}
	err = rows.Err()
	if err != nil {
		slog.Error(`db_row_scan_er`, `error`, err)
		return
	}

	var instanceVersion string
	err = db.QueryRow("SELECT version FROM V$INSTANCE").Scan(&instanceVersion)
	if err != nil {
		slog.Error(`querying_version_er`, `error`, err)
		return
	}
	fmt.Printf("\nConcise Instance Version: %s\n", instanceVersion)

	productRows, err := db.Query("SELECT product, version, status FROM PRODUCT_COMPONENT_VERSION WHERE product LIKE 'Oracle Database%'")
	if err != nil {
		slog.Error(`querying_component_er`, `error`, err)
		return
	}
	defer productRows.Close()
	fmt.Println("\nProduct Component Version:")
	for productRows.Next() {
		var product, version, status string
		if err := productRows.Scan(&product, &version, &status); err != nil {
			slog.Error(`scanning_product_component_row_er`, `error`, err)
			return
		}
		fmt.Printf("Product: %s\nVersion: %s\nStatus: %s\n", product, version, status)
	}
	if err = productRows.Err(); err != nil {
		slog.Error(`scanning_product_component_row_er`, `error`, err)
		return
	}

	newUsername := `"user[2]admin"`        // Double-quoted for special characters
	newPassword := `"pass[2]special_char"` // Double-quoted for special characters
	if q == newUsername {
		return
	}

	dropUserSQL := fmt.Sprintf(`DROP USER %s CASCADE`, newUsername)
	fmt.Printf("Attempting to drop user %s if it exists...\n", newUsername)
	_, err = db.Exec(dropUserSQL)
	if err != nil {
		slog.Error(`drop_user_er`, `error`, `newUsername`, newUsername, err)
		return
	}
	fmt.Printf("User %s dropped successfully (if it existed).\n", newUsername)

	// --- 2. Create the New User ---
	// The IDENTIFIED BY clause should use the exact password string.
	// No need for quotes around the password in the SQL string itself if it's a literal,
	// but since our Go variable `newPassword` already contains the necessary double quotes,
	// we use it directly.
	createUserSQL := fmt.Sprintf(`CREATE USER %s IDENTIFIED BY %s`, newUsername, newPassword)
	fmt.Printf("Creating user %s...\n", newUsername)
	_, err = db.Exec(createUserSQL)
	if err != nil {
		slog.Error(`creating_user_er`, `newUsername`, newUsername, `error`, err)
		return
	}
	fmt.Printf("User %s created successfully.\n", newUsername)

	// --- 3. Grant Roles and Privileges ---
	// Granting CONNECT and RESOURCE roles, and the DBA role for admin access.
	// The DBA role is very powerful.
	grantSQL := fmt.Sprintf(`GRANT CONNECT, RESOURCE, DBA TO %s`, newUsername)
	fmt.Printf("Granting roles and privileges to %s...\n", newUsername)
	_, err = db.Exec(grantSQL)
	if err != nil {
		slog.Error(`granting_privileges_er`, `newUsername`, newUsername, `error`, err)
		return
	}
	fmt.Printf("Roles and privileges granted to %s successfully.\n", newUsername)

	fmt.Printf("\nNew admin user '%s' with password '%s' created successfully.\n", newUsername, newPassword)
}

func doubleQuoted(s string) string {
	return `"` + s + `"`
}

// isOracleUserNameNeedsQuoting checks if Oracle username s needs quoting
func isOracleUserNameNeedsQuoting(s string) bool {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			s = s[1 : len(s)-1]
		}
	}

	if len(s) == 0 || len(s) > 30 {
		return true
	}

	firstRune := []rune(s)[0]
	if !unicode.IsLetter(firstRune) {
		return true
	}

	var validUnquoted = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_$#]{0,29}$`)
	if !validUnquoted.MatchString(s) {
		return true
	}

	// Check reserved words
	upper := strings.ToUpper(s)

	return oracleReservedWords[upper]
}

var oracleReservedWords = map[string]bool{
	"ACCESS":     true,
	"ADD":        true,
	"ALL":        true,
	"ALTER":      true,
	"AND":        true,
	"ANY":        true,
	"AS":         true,
	"ASC":        true,
	"AUDIT":      true,
	"BETWEEN":    true,
	"BY":         true,
	"CHAR":       true,
	"CHECK":      true,
	"CLUSTER":    true,
	"COLUMN":     true,
	"COMMENT":    true,
	"COMPRESS":   true,
	"CONNECT":    true,
	"CREATE":     true,
	"CURRENT":    true,
	"DATE":       true,
	"DECIMAL":    true,
	"DEFAULT":    true,
	"DELETE":     true,
	"DESC":       true,
	"DISTINCT":   true,
	"DROP":       true,
	"ELSE":       true,
	"EXCLUSIVE":  true,
	"EXISTS":     true,
	"FILE":       true,
	"FLOAT":      true,
	"FOR":        true,
	"FROM":       true,
	"GRANT":      true,
	"GROUP":      true,
	"HAVING":     true,
	"IDENTIFIED": true,
	"IMMEDIATE":  true,
	"IN":         true,
	"INCREMENT":  true,
	"INDEX":      true,
	"INITIAL":    true,
	"INSERT":     true,
	"INTEGER":    true,
	"INTERSECT":  true,
	"INTO":       true,
	"IS":         true,
	"LEVEL":      true,
	"LIKE":       true,
	"LOCK":       true,
	"LONG":       true,
	"MAXEXTENTS": true,
	"MINUS":      true,
	"MLSLABEL":   true,
	"MODE":       true,
	"MODIFY":     true,
	"NOAUDIT":    true,
	"NOCOMPRESS": true,
	"NOT":        true,
	"NOWAIT":     true,
	"NULL":       true,
	"NUMBER":     true,
	"OF":         true,
	"OFFLINE":    true,
	"ON":         true,
	"ONLINE":     true,
	"OPTION":     true,
	"OR":         true,
	"ORDER":      true,
	"PCTFREE":    true,
	"PRIOR":      true,
	"PRIVILEGES": true,
	"PUBLIC":     true,
	"RAW":        true,
	"RENAME":     true,
	"RESOURCE":   true,
	"REVOKE":     true,
	"ROW":        true,
	"ROWID":      true,
	"ROWNUM":     true,
	"ROWS":       true,
	"SELECT":     true,
	"SESSION":    true,
	"SET":        true,
	"SHARE":      true,
	"SIZE":       true,
	"SMALLINT":   true,
	"START":      true,
	"SUCCESSFUL": true,
	"SYNONYM":    true,
	"SYSDATE":    true,
	"TABLE":      true,
	"THEN":       true,
	"TO":         true,
	"TRIGGER":    true,
	"UID":        true,
	"UNION":      true,
	"UNIQUE":     true,
	"UPDATE":     true,
	"USER":       true,
	"VALIDATE":   true,
	"VALUES":     true,
	"VARCHAR":    true,
	"VARCHAR2":   true,
	"VIEW":       true,
	"WHENEVER":   true,
	"WHERE":      true,
	"WITH":       true,

	"ANALYZE":       true,
	"ARCHIVE":       true,
	"BINARY_FLOAT":  true,
	"BINARY_DOUBLE": true,
	"BLOB":          true,
	"CLOB":          true,
	"CONTINUE":      true,
	"CURSOR":        true,
	"DATABASE":      true,
	"DATAFILE":      true,
	"DUMP":          true,
	"EXCEPTION":     true,
	"EXIT":          true,
	"FLOAT4":        true,
	"FLOAT8":        true,
	"INDICATOR":     true,
	"LANGUAGE":      true,
	"LARGE":         true,
	"LONGRAW":       true,
	"MATERIALIZED":  true,
	"NCHAR":         true,
	"NCLOB":         true,
	"NESTED_TABLE":  true,
	"NROWID":        true,
	"NVARCHAR2":     true,
	"PACKAGE":       true,
	"PRAGMA":        true,
	"PROCEDURE":     true,
	"REPLACE":       true,
	"RETURN":        true,
	"SAMPLE":        true,
	"SEQUENCE":      true,
	"TABLESPACE":    true,
	"TYPE":          true,
	"UNDER":         true,
	"UNLIMITED":     true,
	"VALUE":         true,
	"VARRAY":        true,
	"XMLTYPE":       true,
}

// CC=musl-gcc CGO_ENABLED=1 GOOS=linux GOAMD64=v2 go build -ldflags '-linkmode external -extldflags "-static -s -w"' -tags musl -trimpath=true -o CI_PROJECT_DIR/cmd/server CI_PROJECT_DIR/cmd/
// CGO_ENABLED=1 GOOS=linux GOAMD64=v2 go build -a -installsuffix cgo -trimpath=true -o CI_PROJECT_DIR/cmd/server CI_PROJECT_DIR/cmd/
func AMD64Level() string {
	level := `v1`
	if cpuid.CPU.Supports(cpuid.SSE3, cpuid.SSSE3, cpuid.SSE4, cpuid.SSE42, cpuid.POPCNT, cpuid.LAHF) {
		level = `v2`
	}
	if cpuid.CPU.Supports(cpuid.AVX, cpuid.AVX2, cpuid.BMI1, cpuid.BMI2, cpuid.FMA3, cpuid.LZCNT, cpuid.MOVBE, cpuid.F16C) {
		level = `v3`
	}
	if cpuid.CPU.Supports(cpuid.AVX512F, cpuid.AVX512BW, cpuid.AVX512CD, cpuid.AVX512DQ, cpuid.AVX512VL) {
		level = `v4`
	}
	return level
}
