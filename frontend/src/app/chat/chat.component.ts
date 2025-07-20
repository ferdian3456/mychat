import { Component, OnInit, OnDestroy } from '@angular/core';
import { ApiService } from "../api.service";
import { Router, ActivatedRoute } from "@angular/router";
import { WebsocketService } from '../websocket.service';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-chat',
  templateUrl: './chat.component.html',
  styleUrl: './chat.component.css'
})
export class ChatComponent implements OnInit, OnDestroy {
  showSidebar = false;
  users: any[] = [];
  message: any = [];
  inputText: string = '';
  conversation_id: any;
  userChat: any;
  myUser: any;
  myToken: string = '';
  errorMessage: string = '';

  resp: any;

  // WebSocket message subscription
  private messageSub: Subscription | null = null;
  private routeSub: Subscription | null = null;

  constructor(
    public api: ApiService,
    public router: Router,
    private route: ActivatedRoute,
    private websocketService: WebsocketService
  ) {}

  ngOnInit() {
    // First, get the WebSocket token and connect
    this.api.getWebsocketToken('/api/ws-token').subscribe(
      (resp) => {
        this.resp = resp
        this.myToken = this.resp.data.websocket_token;
        this.websocketService.connect(this.myToken);
      },
      (error) => this.handleError(error, true)
    );

    // Listen for route changes (conversation ID changes)
    this.routeSub = this.route.paramMap.subscribe(params => {
      const id = params.get('id');
      if (id) {
        this.loadConversationData(id);
      }

      // Also load all sidebar users once
      this.api.getAllUserInfo('api/users').subscribe(
        (resp) => {
          this.resp = resp
          this.users = this.resp.data;
        },
        (error) => this.handleError(error)
      );
    });
  }

  loadConversationData(conversation_id: string) {
  this.conversation_id = conversation_id;

  // 🔥 Reset old data BEFORE any async calls
  this.message = [];
  this.userChat = null;

  // ❌ Don't wait until WebSocket subscription to clear messages — do it now

  // 🔌 Unsubscribe from old stream
  if (this.messageSub) {
    this.messageSub.unsubscribe();
  }

  const participantUrl = `api/conversation/${this.conversation_id}/participant`;
  this.api.getAllParticipantInfo(participantUrl).subscribe(
    (resp) => {
      this.resp = resp
      this.userChat = this.resp.data;

      this.api.getUserInfo('api/userinfo').subscribe(
        (resp) => {
          this.resp = resp
          this.myUser = this.resp.data;

          const msgUrl = `api/conversation/${this.conversation_id}/messages`;

          // ⏳ Get messages for new conversation
          this.api.getAllPastMessages(msgUrl).subscribe(
            (resp) => {
              this.resp = resp
              this.message = this.resp.data.reverse(); // ⏮️ Reverse if needed
            },
            (error) => this.handleError(error)
          );

          // 🎯 Resubscribe with fresh conversation ID
          this.messageSub = this.websocketService.messages$.subscribe((newMsg) => {
            if (Number(newMsg.conversation_id) === Number(this.conversation_id)) {
              this.message.push(newMsg);
            }
          });

        },
        (error) => this.handleError(error)
      );
    },
    (error) => this.handleError(error)
  );
}


  sendMessage(text: string) {
    const msg = {
      conversation_id: Number(this.conversation_id),
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
      (resp) => {
        this.resp = resp
        const newConversationId = this.resp.data.conversation_id;
        this.router.navigate(['/chat', newConversationId]);
      },
      (error) => this.handleError(error)
    );
  }

  handleError(error: any, redirectToLogin: boolean = false) {
    const errorData = error.error?.data;
    if (errorData && typeof errorData === 'object') {
      const firstKey = Object.keys(errorData)[0];
      this.errorMessage = errorData[firstKey];
    } else {
      this.errorMessage = "An unexpected error occurred.";
    }

    console.error(this.errorMessage);
    if (redirectToLogin) {
      this.router.navigate(['/login']);
    }
  }

  ngOnDestroy() {
    if (this.messageSub) {
      this.messageSub.unsubscribe();
    }
    if (this.routeSub) {
      this.routeSub.unsubscribe();
    }
  }

  handleEnterKey() {
    if (this.inputText.trim()) {
      this.sendMessage(this.inputText.trim());
      this.inputText = '';
    }
  }
}
