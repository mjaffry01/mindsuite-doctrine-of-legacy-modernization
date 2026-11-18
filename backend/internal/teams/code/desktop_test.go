package code

import (
	"context"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestDesktopApp runs the team on a small VB.NET Windows Forms app inside a GitHub zip folder:
// SQL in multi-line strings, its own login with an unsalted SHA-256 hash, SQL Server LocalDB,
// layer folders inside one project, screens, and a database file that must not be read.
func TestDesktopApp(t *testing.T) {
	root := "shop-1a2b3c4d/"
	files := map[string][]byte{
		root + "README.md":        []byte("# Shop\n"),
		root + "Shop/Shop.vbproj": []byte("<Project Sdk=\"Microsoft.NET.Sdk\"><PropertyGroup><TargetFramework>net48</TargetFramework></PropertyGroup></Project>\n"),
		root + "Shop/ShopDB.mdf":  []byte("binary"),
		root + "Shop/Data/Db.vb":  []byte("Public Class Db\n    Private Shared ReadOnly cs As String = $\"Data Source=(LocalDB)\\MSSQLLocalDB;AttachDbFilename={p};Integrated Security=True\"\nEnd Class\n"),
		root + "Shop/Data/ItemRepository.vb": []byte(`Public Class ItemRepository
    Public Sub Add()
        Dim sql As String = "
            INSERT INTO Items (Name)
            VALUES (@Name)
        "
    End Sub
    Public Sub Rename()
        Dim sql As String = "
            UPDATE Items
            SET Name = @Name
            WHERE Id = @Id
        "
    End Sub
    Public Function Find() As String
        ' a comment with a quote " does not open a string
        Dim sql As String = "
            SELECT i.Name
            FROM Items i
            INNER JOIN Shelves s ON i.ShelfId = s.Id
        "
        Return "SELECT 1"
    End Function
End Class
`),
		root + "Shop/Data/UserRepository.vb": []byte(`Public Class UserRepository
    Public Function Login(u As String, p As String) As Boolean
        Dim h As String = PasswordHelper.HashPassword(p)
        Dim sql As String = "
            SELECT COUNT(*)
            FROM Users
            WHERE Username = @u AND PasswordHash = @h
        "
    End Function
End Class
`),
		root + "Shop/Helpers/PasswordHelper.vb": []byte(`Imports System.Security.Cryptography
Public Class PasswordHelper
    Public Shared Function HashPassword(password As String) As String
        Using sha As SHA256 = SHA256.Create()
            Return Convert.ToBase64String(sha.ComputeHash(Encoding.UTF8.GetBytes(password)))
        End Using
    End Function
End Class
`),
		root + "Shop/Forms/LoginForm.Designer.vb": []byte("<Global.Microsoft.VisualBasic.CompilerServices.DesignerGenerated()>\nPartial Class LoginForm\n    Inherits System.Windows.Forms.Form\nEnd Class\n"),
		root + "Shop/Forms/ItemForm.Designer.vb":  []byte("Partial Class ItemForm\n    Inherits System.Windows.Forms.Form\nEnd Class\n"),
		root + "Shop/Services/ItemService.vb":     []byte("Public Class ItemService\nEnd Class\n"),
	}
	run, err := agents.NewRun(agents.LoadFiles("shop", files), agents.Options{}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	run.Execute(context.Background())
	snap := run.Snapshot()
	titles := map[string][]string{}
	for _, f := range snap.Findings {
		if f.Status != StatusRejected {
			titles[f.Kind] = append(titles[f.Kind], f.Title)
		}
	}
	has := func(kind, sub string) {
		t.Helper()
		for _, x := range titles[kind] {
			if strings.Contains(x, sub) {
				return
			}
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles[kind])
	}
	has("table-access", "Shop writes ITEMS (2 places)")
	has("table-access", "Shop reads ITEMS")
	has("table-access", "Shop reads SHELVES")
	has("table-access", "Shop reads USERS")
	has("auth-method", "Database-stored passwords")
	has("weak-crypto", "Weak password storage: SHA256")
	has("connection", `SQL Server connection: (LocalDB)\MSSQLLocalDB`)
	has("module", "Module Shop:")
	has("architecture-style", "Single module, layered inside")
	has("screen", "Desktop screen LoginForm")
	has("screen", "Desktop screen ItemForm")
	for _, x := range titles["module"] {
		if strings.Contains(x, "shop-1a2b3c4d") {
			t.Errorf("the zip's top folder became a module: %s", x)
		}
	}
	if len(snap.Questions) != 0 {
		t.Errorf("no login question expected, got %v", snap.Questions)
	}
	skipped := false
	for _, s := range snap.Coverage.Skipped {
		if strings.HasSuffix(s.Path, "ShopDB.mdf") && strings.HasPrefix(s.Reason, "database file") {
			skipped = true
		}
	}
	if !skipped {
		t.Errorf("the .mdf should be skipped as a database file: %+v", snap.Coverage.Skipped)
	}
}
