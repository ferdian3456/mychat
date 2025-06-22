import { Component } from '@angular/core';
import { ApiService } from "../api.service";
import { Router } from "@angular/router";

@Component({
  selector: 'app-register',
  templateUrl: './register.component.html',
  styleUrls: ['./register.component.css']
})
export class RegisterComponent {
  data: any = {
    username: "",
    password: ""
  };

  resp:any

  constructor(
    public api: ApiService,
    public router: Router,
  ) {}

  loading = false;
  errorMessage = '';
  
  validateFields(): boolean {
    const { username, password } = this.data;

    if (!username || username.length < 4 || username.length > 22) {
      this.errorMessage = 'Username must be 4-22 characters.';
      return false;
    }

    if (!password || password.length < 5 || password.length > 20  ) {
      this.errorMessage = 'Password must be 5-20 characters.';
      return false;
    }

    return true;
  }

  doRegister() {
    this.errorMessage = '';
    this.loading = true;

    const isValid = this.validateFields();

    if (!isValid) {
      // Delay hiding loader so spinner can render
      setTimeout(() => {
        this.loading = false;
      }, 300); // 300ms is usually enough
      return;
    }

    // Proceed with login
    this.api.register('register', this.data).subscribe(
      (resp) => {
        this.loading = false;
        console.log(this.resp)
        this.resp = resp;
        // 👇 Send a request to check if the cookie/session is valid
        this.api.getUserInfo('api/userinfo').subscribe(
          () => {
            // ✅ Cookie is valid, now allow navigation
            this.router.navigate(['/chat']);
          },
          () => {
            this.errorMessage = 'Login succeeded, but session check failed.';
          }
        );
      },
      (error) => {
        this.loading = false;

        // Attempt to extract the first error message dynamically
        const errorData = error.error?.data;
        if (errorData && typeof errorData === 'object') {
          const firstKey = Object.keys(errorData)[0]; // Get the first key, e.g., "username" or "database"
          this.errorMessage = errorData[firstKey];
        } else {
          this.errorMessage = "An unexpected error occurred.";
        }
      }
    );
  }
}
