import { Component,OnInit } from '@angular/core';
import { ApiService } from "../api.service";
import { Router } from "@angular/router";

@Component({
  selector: 'app-chat-list',
  templateUrl: './chat-list.component.html',
  styleUrl: './chat-list.component.css'
})
export class ChatListComponent {
  showSidebar = false;

  conversation_id:any
  users: any[]=[]
  errorMessage : any
  request :any ={
    participant_ids:[] as string[],
  }

  resp:any

  constructor(
    public api: ApiService,
    public router: Router,
  ) {}

  ngOnInit(){
    this.api.getAllUserInfo('api/users').subscribe(
      (resp) => {
        this.resp = resp;
        this.users = this.resp.data
        console.log(this.users)
      },
      (error) => {
        const errorData = error.error?.data;
        if (errorData && typeof errorData === 'object') {
          const firstKey = Object.keys(errorData)[0]; // Get the first key, e.g., "username" or "database"
          this.errorMessage = errorData[firstKey];
          console.log(this.errorMessage)
          this.router.navigate(['/login'])
        } else {
          this.errorMessage = "An unexpected error occurred.";
          console.log(this.errorMessage)
          this.router.navigate(['/login'])
        }
      }
    );
  }

  goToChat(userid: string) {
    this.request.participant_ids = [userid]
    this.api.createOrGetConversation('api/conversation', this.request).subscribe(
      (resp)=>{
        this.resp = resp
        this.conversation_id = this.resp.data.conversation_id
        this.router.navigate(['/chat/', this.conversation_id]);
      },
      (error) => {
        const errorData = error.error?.data;
        if (errorData && typeof errorData === 'object') {
          const firstKey = Object.keys(errorData)[0]; // Get the first key, e.g., "username" or "database"
          this.errorMessage = errorData[firstKey];
          console.log(this.errorMessage)
        } else {
          this.errorMessage = "An unexpected error occurred.";
          console.log(this.errorMessage)
        }
      }
    )
  }
}
