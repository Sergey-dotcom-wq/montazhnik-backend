package main
import ("fmt"; "os"; "golang.org/x/crypto/bcrypt")
func main() {
    h := os.Args[1]
    for _, p := range []string{"Pass123456", "Pass123456\n", " Pass123456", "Pass123456 "} {
        err := bcrypt.CompareHashAndPassword([]byte(h), []byte(p))
        fmt.Printf("%q -> %v\n", p, err == nil)
    }
}
