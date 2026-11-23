"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useForm } from "react-hook-form";

import { FormAlert } from "@/components/auth/AuthLayout";
import { useAuthRedirect } from "@/components/auth/use-auth-redirect";
import { useRegister } from "@/hooks/use-auth";
import { registerSchema, type RegisterValues } from "@/lib/auth-schemas";
import { errorMessage, isApiError } from "@/lib/errors";

const fields = ["displayName", "email", "password"] as const;
type Field = (typeof fields)[number];
const isField = (f: string | undefined): f is Field => fields.includes(f as Field);

export function RegisterForm() {
  const { returnTo, goNext } = useAuthRedirect();
  const signUp = useRegister();
  const { register, handleSubmit, setError, formState: { errors } } = useForm<RegisterValues>({ resolver: zodResolver(registerSchema) });

  // Registration already starts the session (201 + cookie), so no separate login.
  const onSubmit = handleSubmit(({ displayName, email, password }) =>
    signUp.mutate({ displayName, email, password }, {
      onSuccess: goNext,
      onError: (error) => {
        if (isApiError(error) && isField(error.field)) {
          setError(error.field, { message: error.code === "CONFLICT" ? "This email is already registered. Sign in instead." : error.message });
        }
      },
    }),
  );
  const formError = signUp.error && !(isApiError(signUp.error) && isField(signUp.error.field)) ? signUp.error : null;
  const loginHref = returnTo === "/dashboard" ? "/login" : `/login?returnTo=${encodeURIComponent(returnTo)}`;

  return (
    <>
      <form onSubmit={onSubmit} noValidate>
        {formError && <FormAlert message={errorMessage(formError)} requestId={isApiError(formError) ? formError.requestId : undefined} />}
        <label>
          Display name
          <input autoComplete="nickname" placeholder="How other readers see you" aria-invalid={!!errors.displayName} {...register("displayName")} />
          {errors.displayName && <span className="field-error">{errors.displayName.message}</span>}
        </label>
        <label>
          Email address
          <input type="email" autoComplete="email" placeholder="you@example.com" aria-invalid={!!errors.email} {...register("email")} />
          {errors.email && <span className="field-error">{errors.email.message}</span>}
        </label>
        <label>
          Password
          <input type="password" autoComplete="new-password" placeholder="At least 8 characters" aria-invalid={!!errors.password} {...register("password")} />
          {errors.password && <span className="field-error">{errors.password.message}</span>}
        </label>
        <label>
          Confirm password
          <input type="password" autoComplete="new-password" placeholder="Type it again" aria-invalid={!!errors.confirmPassword} {...register("confirmPassword")} />
          {errors.confirmPassword && <span className="field-error">{errors.confirmPassword.message}</span>}
        </label>
        <button className="button button-primary" type="submit" disabled={signUp.isPending}>{signUp.isPending ? "Creating your account…" : "Create account →"}</button>
      </form>
      <p className="login-switch">Already have an account? <Link href={loginHref}>Sign in</Link></p>
    </>
  );
}
