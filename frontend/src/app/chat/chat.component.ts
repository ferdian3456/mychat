import { Component,OnInit } from '@angular/core';
import { ApiService } from "../api.service";
import { Router } from "@angular/router";
import { ActivatedRoute } from '@angular/router';
import { WebsocketService } from '../websocket.service'; 


@Component({
  selector: 'app-chat',
  templateUrl: './chat.component.html',
  styleUrl: './chat.component.css'
})
export class ChatComponent {
  showSidebar = false;
  users: any[]=[]

  conversation_id:any
  userChat:any
  resp:any
  myToken: any = [] 
  myUser: any
  inputText: string = '';
  message: any = []
 
  errorMessage : any

  constructor(
    public api: ApiService,
    public router: Router,
    private route: ActivatedRoute,
    private websocketService: WebsocketService // inject service
  ) {}

  ngOnInit(){
    this.api.getWebsocketToken('/api/ws-token').subscribe(
      (resp) => {
        this.resp = resp
        this.myToken = this.resp.data.websocket_token
        //console.log(this.myToken)
        this.websocketService.connect(this.myToken);

        this.websocketService.messages$.subscribe((newMsg) => {
          console.log("new message from websocket", newMsg);

          // 🛠 Only push if it's for the current conversation
          if (Number(newMsg.conversation_id) === Number(this.conversation_id)) {
            this.message.push(newMsg);
          }
        });
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
    )

    this.conversation_id = this.route.snapshot.paramMap.get('id')!;
    const url = `api/conversation/${this.conversation_id}/participant`;
    this.api.getAllParticipantInfo(url).subscribe(
      (resp) => {
        this.resp = resp
        this.userChat = this.resp.data
        console.log(this.resp.data)

        this.api.getUserInfo('api/userinfo').subscribe(
          (resp) => {
            this.resp = resp
            this.myUser = this.resp.data
            const url2 = `api/conversation/${this.conversation_id}/messages`;
            this.api.getAllPastMessages(url2).subscribe(
              (resp) => {
                this.resp = resp
                this.message = this.resp.data.reverse()
                console.log("message",this.message)
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
        } else {
          this.errorMessage = "An unexpected error occurred.";
          console.log(this.errorMessage)
        }
      }
    );
  }

  sendMessage(text: string) {
    const msg = {
      conversation_id: Number(this.conversation_id), // ✅ ensure it's a number
      text: text,
      sender_id: this.myUser.id,
    };
    this.websocketService.sendMessage(msg);
  }

  goToChat(userid: string) {
    const data = {
      participant_ids: [userid]
    };
    this.api.createOrGetConversation('api/conversation', data).subscribe(
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


