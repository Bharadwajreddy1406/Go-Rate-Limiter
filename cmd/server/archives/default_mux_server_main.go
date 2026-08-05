package archives
import (
	"fmt"
	"net/http"
)

func SlashHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Request URL ==> %s Method ==> %s\n", r.URL.Path, r.Method)
	fmt.Fprintf(w, "Hello, World!")
}


func main() {
	
	http.HandleFunc("/hello", SlashHandler)

	fmt.Println("Server is running on port", "9090")

	err := http.ListenAndServe(":9090", nil)

	if err != nil {
		fmt.Println("Error starting server:", err)
	}

}