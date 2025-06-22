import { Injectable } from '@angular/core';
import { CanActivate, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';

@Injectable({
  providedIn: 'root',
})
export class AuthGuard implements CanActivate {
  constructor(private http: HttpClient, private router: Router) {}

  canActivate(): Observable<boolean> {
    return this.http.get('http://localhost:8090/api/userinfo', { withCredentials: true }).pipe(
      map(() => {
        // If request succeeds, allow navigation
        return true;
      }),
      catchError((err) => {
        // If request fails, redirect to login
        this.router.navigate(['/login']);
        return of(false);
      })
    );
  }
}
