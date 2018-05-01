package main

/*
#include <stdlib.h>

typedef struct Request_
{
    const char *Method;
    const char *Host;
	const char *URL;
	const char *Body;
    const char *Headers;
} Request;

typedef struct RequestFastHttp_
{
    void *Method;
    int MethodLen;
    void *Host;
    int HostLen;
	void *URL;
	int URLLen;
	void *Body;
	int BodyLen;
    void *Headers;
    int HeadersLen;
} RequestFastHttp;

typedef unsigned int ResponseWriterPtr;

typedef void FuncPtr(ResponseWriterPtr w, Request *r);

extern void Call_HandleFunc(ResponseWriterPtr w, Request *r, FuncPtr *fn);
extern void Call_HandleFuncFastHttp(ResponseWriterPtr w, RequestFastHttp *r, FuncPtr *fn);
*/
import "C"
import (
	"bytes"
	"net/http"
	"unsafe"
	"github.com/valyala/fasthttp"
	"sync"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var cpointers = PtrProxy()

func listenAndServeNetHttp(caddr *C.char) {
	addr := C.GoString(caddr)
	http.ListenAndServe(addr, nil)
}

//export ListenAndServe
func ListenAndServe(caddr *C.char) {
	go func() {
		log.Println("Listening signals...")
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		//errc <- fmt.Errorf("Signal %v", <-c)
	}()
	listenAndServeFastHttp(caddr)
}

var handlerMap = make(map[string]func(*fasthttp.RequestCtx))
var handlerMutex = sync.RWMutex{}

func handlerFastHttp(ctx *fasthttp.RequestCtx) {
	//handlerMutex.RLock()
	recvCtx, ok := handlerMap[string(ctx.Path())]
	if ok {
		recvCtx(ctx)
	} else {
		fmt.Printf("Path not found: \"%s\"\n", string(ctx.Path()))
	}
	//handlerMutex.RUnlock()
}

func listenAndServeFastHttp(caddr *C.char) {
	addr := C.GoString(caddr)
	fasthttp.ListenAndServe(addr, handlerFastHttp)
}

func handleFuncHttp(cpattern *C.char, cfn *C.FuncPtr) {
	// C-friendly wrapping for our http.HandleFunc call.
	pattern := C.GoString(cpattern)
	http.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
		// Convert the headers to a String
		headerBuffer := new(bytes.Buffer)
		req.Header.Write(headerBuffer)
		headersString := headerBuffer.String()
		// Convert the request body to a String
		bodyBuffer := new(bytes.Buffer)
		bodyBuffer.ReadFrom(req.Body)
		bodyString := bodyBuffer.String()
		// Wrap relevant request fields in a C-friendly datastructure.
		creq := C.Request{
			Method:  C.CString(req.Method),
			Host:    C.CString(req.Host),
			URL:     C.CString(req.URL.String()),
			Body:    C.CString(bodyString),
			Headers: C.CString(headersString),
		}
		// Convert the ResponseWriter interface instance to an opaque C integer
		// that we can safely pass along.
		wPtr := cpointers.Ref(unsafe.Pointer(&w))
		// Call our C function pointer using our C shim.
		C.Call_HandleFunc(C.ResponseWriterPtr(wPtr), &creq, cfn)
		// release the C memory
		C.free(unsafe.Pointer(creq.Method))
		C.free(unsafe.Pointer(creq.Host))
		C.free(unsafe.Pointer(creq.URL))
		C.free(unsafe.Pointer(creq.Body))
		C.free(unsafe.Pointer(creq.Headers))
		// Release the ResponseWriter from the registry since we're done with
		// this response.
		cpointers.Free(wPtr)
	})
}

func handleFuncFastHttp(cpattern *C.char, cfn *C.FuncPtr) {
	pattern := C.GoString(cpattern)
	//fmt.Printf("Register pattern: \"%s\"\n", string(pattern))
	handlerMutex.Lock()
	handlerMap[string(pattern)] = func(ctx *fasthttp.RequestCtx) {
		w := bytes.Buffer{}

		// Convert the ResponseWriter interface instance to an opaque C integer
		// that we can safely pass along.
		wPtr := cpointers.Ref(unsafe.Pointer(&w))

		/*creq := C.Request{
			Method:  C.CString(string(ctx.Method())),
			Host:    C.CString(string(ctx.Host())),
			URL:     C.CString(string(ctx.URI().FullURI())),
			Body:    C.CString(string(ctx.PostBody())),
			Headers: C.CString(string(ctx.Request.Header.Header())),
		}*/

		creq := C.RequestFastHttp{
			Method:  C.CBytes(ctx.Method()),
			MethodLen:  C.int(len(ctx.Method())),
			Host:    C.CBytes(ctx.Host()),
			HostLen:    C.int(len(ctx.Host())),
			URL:     C.CBytes(ctx.URI().FullURI()),
			URLLen:     C.int(len(ctx.URI().FullURI())),
			Body:    C.CBytes(ctx.PostBody()),
			BodyLen:    C.int(len(ctx.PostBody())),
			Headers: C.CBytes(ctx.Request.Header.Header()),
			HeadersLen: C.int(len(ctx.Request.Header.Header())),
		}


		// Call our C function pointer using our C shim.
		//C.Call_HandleFunc(C.ResponseWriterPtr(wPtr), &creq, cfn)
		C.Call_HandleFuncFastHttp(C.ResponseWriterPtr(wPtr), &creq, cfn)
		// release the C memory
		C.free(unsafe.Pointer(creq.Method))
		C.free(unsafe.Pointer(creq.Host))
		C.free(unsafe.Pointer(creq.URL))
		C.free(unsafe.Pointer(creq.Body))
		C.free(unsafe.Pointer(creq.Headers))
		// Release the ResponseWriter from the registry since we're done with
		// this response.
		cpointers.Free(wPtr)
		ctx.Write(w.Bytes())
	}
	handlerMutex.Unlock()
}

//export HandleFunc
func HandleFunc(cpattern *C.char, cfn *C.FuncPtr) {
	handleFuncFastHttp(cpattern, cfn)
}

func main() {}
