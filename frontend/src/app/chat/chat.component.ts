import { Component, OnInit, OnDestroy, ViewChild, ElementRef } from '@angular/core';
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
  @ViewChild('messagesContainer') private messagesContainer!: ElementRef;
  @ViewChild('usernameInput') usernameInputRef!: ElementRef;
  @ViewChild('chatInput') chatInputRef!: ElementRef;
  showSidebar = false;
  users: any[] = [];
  message: any = [];
  inputText: string = '';
  conversation_id: any;
  userChat: any;
  myUser: any;
  myToken: string = '';
  errorMessage: string = '';
  showAddUserModal = false;
  loading = false;
  usernameErrorMessage: any
  newUsername: string = '';
  dataAdd: any = {
    username: "",
   };

  resp: any;

  private isUserNearBottom(): boolean {
  const threshold = 100; // px from bottom
  const container = this.messagesContainer.nativeElement;
  const position = container.scrollTop + container.clientHeight;
  const height = container.scrollHeight;
  return position > height - threshold;
  }

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

      this.api.getAllConversation('api/conversation').subscribe(
          (resp) => {
            this.resp = resp;
            this.users = this.resp.data 
            console.log(this.resp)
          },
          (error) => {
            this.handleError(error);
          }
        );
      },
    );
  }

  onAddUser() {
    this.newUsername = '';
    this.showAddUserModal = true;

    // Give the modal time to render before focusing
    setTimeout(() => {
      this.usernameInputRef?.nativeElement.focus();
    }, 100);
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
              this.resp = resp;
              this.message = this.resp.data.reverse();
              setTimeout(() => {
                this.scrollToBottom();
                this.chatInputRef?.nativeElement.focus();  // 👈 Focus here
              }, 100);
            },
            (error) => this.handleError(error)
          );
          // 🎯 Resubscribe with fresh conversation ID
          this.messageSub = this.websocketService.messages$.subscribe((newMsg) => {
            console.log("WebSocket message received:", newMsg);
            console.log("created_at raw:", newMsg.created_at);
            console.log("Parsed as:", new Date(newMsg.created_at))
          if (Number(newMsg.conversation_id) === Number(this.conversation_id)) {
            const shouldAutoScroll = this.isUserNearBottom();
            this.message.push(newMsg);

            if (shouldAutoScroll) {
              setTimeout(() => this.scrollToBottom(), 100);
            }
          }
        });
        },
        (error) => this.handleError(error)
      );
    },
    (error) => this.handleError(error)
  );
}

  shouldShowDateSeparator(index: number): boolean {
  if (index === 0) return true;

  const currentDate = this.getDatePart(this.message[index].created_at);
  const prevDate = this.getDatePart(this.message[index - 1].created_at);

  return currentDate !== prevDate;
  }

  getDatePart(dateString: string): string {
    // Extract just the date portion (YYYY-MM-DD)
    const match = dateString.match(/^(\d{4}-\d{2}-\d{2})/);
    return match ? match[1] : '';
  }

  formatDateLabel(dateStr: string): string {
    // Convert "2025-07-24" to "Jul 24, 2025"
    const [year, month, day] = dateStr.split('-');
    const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 
                        'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    return `${monthNames[Number(month) - 1]} ${day}, ${year}`;
  }

  getLocalTimeString(dateString: string): string {
  // Match HH:MM from ISO string like "2025-07-24T17:50:00.000000+07:00"
  const match = dateString.match(/T(\d{2}):(\d{2})/);
  if (match) {
    const [, hours, minutes] = match;
    return `${hours}:${minutes}`;
  }
  return 'Invalid time';
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
    console.log(userid)
    this.router.navigate(['/chat/', userid])
    // this.api.createOrGetConversation('api/conversation', this.request).subscribe(
    //   (resp) => {
    //     this.resp = resp;
    //     this.conversation_id = this.resp.data.conversation_id;
    //     this.router.navigate(['/chat/', this.conversation_id]);
    //   },
    //   (error) => {
    //     this.handleError(error);
    //   }
    // );
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


  validateFields(username:string): boolean {
    if (!username || username.length < 4 || username.length > 22) {
      this.usernameErrorMessage = 'Username must be 4-22 characters.';
      return false;
    }

    return true;
  }

   startChatWithUsername() {
    const enteredUsername = this.newUsername.trim();
    this.usernameErrorMessage = '';

    this.loading = true;
    const isValid = this.validateFields(enteredUsername);
    if (!isValid) {
      // Delay hiding loader so spinner can render
      setTimeout(() => {
        this.loading = false;
      }, 300); // 300ms is usually enough
      return;
    }

    this.dataAdd.username = enteredUsername;
    console.log(this.dataAdd)

    this.api.createOrGetConversation('api/conversation', this.dataAdd).subscribe(
      (resp) => {
        this.loading = false;
        this.resp = resp;
        console.log(this.resp)
        window.location.reload()
      },
      (error) => {
         this.loading = false;

        // Attempt to extract the first error message dynamically
        const errorData = error.error?.data;
        if (errorData && typeof errorData === 'object') {
          const firstKey = Object.keys(errorData)[0]; // Get the first key, e.g., "username" or "database"
          this.usernameErrorMessage = errorData[firstKey];
        } else {
          this.usernameErrorMessage = "An unexpected error occurred.";
        }
      }
    );
  }

  private scrollToBottom(): void {
  try {
    this.messagesContainer.nativeElement.scrollTop = this.messagesContainer.nativeElement.scrollHeight;
  } catch (err) {
    console.error('Scroll failed', err);
  }
}
}
