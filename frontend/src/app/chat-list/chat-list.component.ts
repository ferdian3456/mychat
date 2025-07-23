import { Component, OnInit } from '@angular/core';
import { ApiService } from '../api.service';
import { Router } from '@angular/router';

@Component({
  selector: 'app-chat-list',
  templateUrl: './chat-list.component.html',
  styleUrls: ['./chat-list.component.css'],
})
export class ChatListComponent implements OnInit {
  showSidebar = false;
  loading = false;

  conversation_id: any;
  users: any[] = [];
  myself: any;
  errorMessage: any;
  usernameErrorMessage: any
  request: any = {
    participant_ids: [] as string[],
  };

  dataAdd: any = {
    username: "",
   };

  resp: any;

  showAddUserModal = false;
  newUsername: string = '';

  constructor(public api: ApiService, public router: Router) {}

  ngOnInit() {
    this.api.getUserInfo('api/userinfo').subscribe(
      (resp) => {
        this.resp = resp;
        this.myself = this.resp.data;

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
      (error) => {
        this.handleError(error);
        this.router.navigate(['/login']);
      }
    );
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

  validateFields(username:string): boolean {
    if (!username || username.length < 4 || username.length > 22) {
      this.usernameErrorMessage = 'Username must be 4-22 characters.';
      return false;
    }

    return true;
  }

  onAddUser() {
    this.newUsername = '';
    this.showAddUserModal = true;
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

  private handleError(error: any) {
    const errorData = error.error?.data;
    if (errorData && typeof errorData === 'object') {
      const firstKey = Object.keys(errorData)[0];
      this.errorMessage = errorData[firstKey];
    } else {
      this.errorMessage = 'An unexpected error occurred.';
    }
    console.error(this.errorMessage);
  }
}
