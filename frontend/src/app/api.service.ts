import { Injectable } from '@angular/core';
import { HttpClient } from "@angular/common/http";

@Injectable({
  providedIn: 'root'
})
export class ApiService {

  baseUrl = "http://localhost:8090/";
  chatUrl = "http://localhost:8091/";

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

  getAllUserInfo(url: string) {
    return this.http.get(this.baseUrl + url, { withCredentials: true });
  }

  getMessagesWithBeforeId(url: string) {
    return this.http.get(this.baseUrl + url, { withCredentials: true });
  }

  getAllConversation(url:string){
    return this.http.get(this.chatUrl + url, { withCredentials: true });
  }

  createOrGetConversation(url:string,data:any) {
    return this.http.post(this.chatUrl + url, data, { withCredentials: true});
  }

  getAllParticipantInfo(url:string){
    return this.http.get(this.chatUrl + url, { withCredentials: true });
  }

  getAllPastMessages(url:string){
      return this.http.get(this.chatUrl + url, { withCredentials: true });
  }

  getWebsocketToken(url:string) {
    return this.http.get(this.chatUrl + url, { withCredentials: true });
  }
}
