#include "_cgo_export.h"

void Call_HandleFunc(ResponseWriterPtr w, Request *req, FuncPtr *fn) {
    return fn(w, req);
}

void Call_HandleFuncFastHttp(ResponseWriterPtr w, RequestFastHttp *req, FuncPtr *fn) {
    // todo: Use the len fields to handle content with null-byte
    Request r;
    r.Method = (const char*)req->Method;
    r.Host = (const char*)req->Host;
    r.URL = (const char*)req->URL;
    r.Body = (const char*)req->Body;
    r.Headers = (const char*)req->Headers;
    return fn(w, &r);
}
