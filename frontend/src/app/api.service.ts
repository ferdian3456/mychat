import { Injectable } from '@angular/core';
import { HttpClient } from "@angular/common/http";

@Injectable({
  providedIn: 'root'
})
export class ApiService {

  baseUrl = "http://localhost:8090/";

  constructor(private http: HttpClient) {}

  login(url: string, data: any) {
    return this.http.post(this.baseUrl + url, data, { withCredentials: true });
  }

  register(url: string, data: any) {
    return this.http.post(this.baseUrl + url, data, { withCredentials: true });
  }

  getUserInfo(url: string) {
    return this.http.get(this.baseUrl + url, { withCredentials: true });
  }
}
