package archives
// import (
// 	"fmt"
// 	"net/http"
// )

// func SlashHandler(w http.ResponseWriter, r *http.Request) {
// 	fmt.Printf("Request URL ==> %s Method ==> %s\n", r.URL.Path, r.Method)
// 	fmt.Fprintf(w, "Hello, World!")
// }


// func main() {
	
// 	http.HandleFunc("/hello", SlashHandler)



// 	fmt.Println("Server is running on port", Port)
// 	mux := http.NewServeMux()

// 	server:= &http.Server{
// 		Addr:    fmt.Sprintf(":%d", Port),
// 		Handler: mux,
// 	}

// 	err := server.ListenAndServe()


	
// 	if err != nil {
// 		fmt.Println("Error starting server:", err)
// 	}

// }