import { Component,OnInit } from '@angular/core';
import { Router } from "@angular/router";
import { ApiService } from "../api.service";

@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrl: './home.component.css'
})
export class HomeComponent {

  resp:any
  myUser:any
  errorMessage:any

  constructor(
      public api: ApiService,
      public router: Router,
  ) {}

  ngOnInit(){
    this.api.getUserInfo('api/userinfo').subscribe(
      (resp) => {
        this.resp = resp
        this.myUser = this.resp.data
        console.log(this.myUser)
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


  goToLogin(){
    this.router.navigate(["/login"])
  }

  goToChat(){
    this.router.navigate(["/chat-list"])
  }
}
