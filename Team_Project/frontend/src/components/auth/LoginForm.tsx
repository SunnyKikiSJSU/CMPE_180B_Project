"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useForm } from "react-hook-form";

import { FormAlert } from "@/components/auth/AuthLayout";
import { useAuthRedirect } from "@/components/auth/use-auth-redirect";
import { useLogin } from "@/hooks/use-auth";
import { loginSchema, type LoginValues } from "@/lib/auth-schemas";
import { errorMessage, isApiError } from "@/lib/errors";

export function LoginForm() {
  const { returnTo, goNext } = useAuthRedirect();
  const login = useLogin();
  const { register, handleSubmit, setError, formState: { errors } } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) });

  const onSubmit = handleSubmit((values) =>
    login.mutate(values, {
      onSuccess: goNext,
      onError: (error) => {
        if (isApiError(error) && (error.field === "email" || error.field === "password")) setError(error.field, { message: error.message });
      },
    }),
  );
  const formError = login.error && !(isApiError(login.error) && (login.error.field === "email" || login.error.field === "password")) ? login.error : null;
  const registerHref = returnTo === "/dashboard" ? "/register" : `/register?returnTo=${encodeURIComponent(returnTo)}`;

  return (
    <>
      <form onSubmit={onSubmit} noValidate>
        {formError && <FormAlert message={errorMessage(formError)} requestId={isApiError(formError) ? formError.requestId : undefined} />}
        <label>
          Email address
          <input type="email" autoComplete="email" placeholder="you@example.com" aria-invalid={!!errors.email} {...register("email")} />
          {errors.email && <span className="field-error">{errors.email.message}</span>}
        </label>
        <label>
          Password
          <input type="password" autoComplete="current-password" placeholder="••••••••" aria-invalid={!!errors.password} {...register("password")} />
          {errors.password && <span className="field-error">{errors.password.message}</span>}
        </label>
        <button className="button button-primary" type="submit" disabled={login.isPending}>{login.isPending ? "Signing in…" : "Sign in →"}</button>
      </form>
      <p className="login-switch">New here? <Link href={registerHref}>Create an account</Link></p>
    </>
  );
}
