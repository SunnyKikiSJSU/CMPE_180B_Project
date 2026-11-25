"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { FormAlert } from "@/components/auth/AuthLayout";
import { ConfirmButton } from "@/components/ConfirmButton";
import { ErrorState } from "@/components/ErrorState";
import { useChangePassword, useDeleteAccount, useProfile, useUpdateProfile } from "@/hooks/use-account";
import { passwordSchema, profileChanges, profileDefaults, profileSchema, type PasswordValues, type ProfileOutput, type ProfileValues } from "@/lib/account-schemas";
import { errorMessage, isApiError } from "@/lib/errors";
import { formatCount, formatDate } from "@/lib/format";
import type { User } from "@/types/domain";

/** /settings: profile, password, and account deletion (GET/PATCH/DELETE /api/users/me, PUT /api/users/me/password). */
export function SettingsView() {
  const profile = useProfile();
  return (
    <div className="section-page settings-page">
      <div className="page-heading">
        <div>
          <p className="eyebrow">Account</p>
          <h1>Settings</h1>
          <p className="heading-copy">Your public profile, your password, and your account.</p>
        </div>
      </div>
      {profile.isPending ? (
        <p className="muted-note" role="status">Loading your account…</p>
      ) : profile.isError ? (
        <section className="panel"><ErrorState error={profile.error} title="Your account did not load" onRetry={() => void profile.refetch()} /></section>
      ) : (
        <div className="settings-grid">
          <ProfileForm user={profile.data} />
          <AccountSummary user={profile.data} />
          <PasswordForm />
          <DeleteAccount />
        </div>
      )}
    </div>
  );
}

function AccountSummary({ user }: { user: User }) {
  const stats = user.stats;
  return (
    <section className="panel settings-card" aria-labelledby="account-heading">
      <h2 id="account-heading">Your account</h2>
      <dl className="settings-facts">
        <div><dt>Email</dt><dd>{user.email}</dd></div>
        <div><dt>Member since</dt><dd>{formatDate(user.createdAt)}</dd></div>
      </dl>
      {stats && (
        <dl className="report-stats compact">
          <div className="report-stat"><dt>Shelves</dt><dd>{formatCount(stats.shelfCount)}</dd></div>
          <div className="report-stat"><dt>Books finished</dt><dd>{formatCount(stats.booksCompleted)}</dd></div>
          <div className="report-stat"><dt>Reviews</dt><dd>{formatCount(stats.reviewsWritten)}</dd></div>
          <div className="report-stat"><dt>Ratings</dt><dd>{formatCount(stats.ratingsGiven)}</dd></div>
        </dl>
      )}
      <p className="muted-note">The email address cannot be changed.</p>
    </section>
  );
}

const profileFields = ["displayName", "avatarUrl", "bio", "favoriteGenre"] as const;
type ProfileField = (typeof profileFields)[number];
const isProfileField = (f: string | undefined): f is ProfileField => profileFields.includes(f as ProfileField);

function ProfileForm({ user }: { user: User }) {
  const update = useUpdateProfile();
  const [notice, setNotice] = useState("");
  const { register, handleSubmit, setError, reset, formState: { errors, isDirty } } = useForm<ProfileValues, unknown, ProfileOutput>({
    resolver: zodResolver(profileSchema),
    defaultValues: profileDefaults(user),
  });
  const onSubmit = handleSubmit((values) => {
    const changes = profileChanges(values, user);
    if (Object.keys(changes).length === 0) {
      setNotice("Nothing changed.");
      return;
    }
    setNotice("");
    update.mutate(changes, {
      onSuccess: (saved) => {
        // Start again from the saved (trimmed) values, so the form is clean.
        reset(profileDefaults(saved));
        setNotice("Profile saved.");
      },
      onError: (error) => {
        if (isApiError(error) && isProfileField(error.field)) setError(error.field, { message: error.message });
      },
    });
  });
  const formError = update.error && !(isApiError(update.error) && isProfileField(update.error.field)) ? update.error : null;
  const field = (name: ProfileField) => ({ "aria-invalid": !!errors[name], ...register(name, { onChange: () => setNotice("") }) });
  return (
    <section className="panel settings-card" aria-labelledby="profile-heading">
      <h2 id="profile-heading">Profile</h2>
      <form className="settings-form" onSubmit={onSubmit} noValidate>
        {formError && <FormAlert message={errorMessage(formError)} requestId={isApiError(formError) ? formError.requestId : undefined} />}
        <label>
          Display name
          <input autoComplete="nickname" maxLength={80} {...field("displayName")} />
          {errors.displayName && <span className="field-error">{errors.displayName.message}</span>}
        </label>
        <label>
          <span>Avatar image address <small>(optional)</small></span>
          <input type="url" inputMode="url" placeholder="https://…" {...field("avatarUrl")} />
          {errors.avatarUrl ? <span className="field-error">{errors.avatarUrl.message}</span> : user.avatarUrl && <small className="muted-note">To change the avatar, enter a new address. It cannot be removed.</small>}
        </label>
        <label>
          <span>Favorite genre <small>(optional)</small></span>
          <input maxLength={80} {...field("favoriteGenre")} />
          {errors.favoriteGenre && <span className="field-error">{errors.favoriteGenre.message}</span>}
        </label>
        <label>
          <span>Bio <small>(optional)</small></span>
          <textarea rows={4} maxLength={2000} {...field("bio")} />
          {errors.bio && <span className="field-error">{errors.bio.message}</span>}
        </label>
        <div className="settings-actions">
          <button className="button button-primary" type="submit" disabled={update.isPending || !isDirty}>{update.isPending ? "Saving…" : "Save profile"}</button>
          {notice && <span className="form-notice" role="status">{notice}</span>}
        </div>
      </form>
    </section>
  );
}

const passwordFields = ["currentPassword", "newPassword"] as const;
type PasswordField = (typeof passwordFields)[number];
const isPasswordField = (f: string | undefined): f is PasswordField => passwordFields.includes(f as PasswordField);

/** A wrong current password is a 400 on currentPassword; it never signs the reader out. */
function PasswordForm() {
  const change = useChangePassword();
  const [saved, setSaved] = useState(false);
  const { register, handleSubmit, setError, reset, formState: { errors } } = useForm<PasswordValues>({
    resolver: zodResolver(passwordSchema),
    defaultValues: { currentPassword: "", newPassword: "", confirmPassword: "" },
  });
  const onSubmit = handleSubmit(({ currentPassword, newPassword }) => {
    setSaved(false);
    change.mutate({ currentPassword, newPassword }, {
      onSuccess: () => {
        reset();
        setSaved(true);
      },
      onError: (error) => {
        if (isApiError(error) && isPasswordField(error.field)) setError(error.field, { message: error.message });
      },
    });
  });
  const formError = change.error && !(isApiError(change.error) && isPasswordField(change.error.field)) ? change.error : null;
  return (
    <section className="panel settings-card" aria-labelledby="password-heading">
      <h2 id="password-heading">Password</h2>
      <form className="settings-form" onSubmit={onSubmit} noValidate>
        {formError && <FormAlert message={errorMessage(formError)} requestId={isApiError(formError) ? formError.requestId : undefined} />}
        <label>
          Current password
          <input type="password" autoComplete="current-password" aria-invalid={!!errors.currentPassword} {...register("currentPassword")} />
          {errors.currentPassword && <span className="field-error">{errors.currentPassword.message}</span>}
        </label>
        <label>
          New password
          <input type="password" autoComplete="new-password" placeholder="At least 8 characters" aria-invalid={!!errors.newPassword} {...register("newPassword")} />
          {errors.newPassword && <span className="field-error">{errors.newPassword.message}</span>}
        </label>
        <label>
          Confirm new password
          <input type="password" autoComplete="new-password" aria-invalid={!!errors.confirmPassword} {...register("confirmPassword")} />
          {errors.confirmPassword && <span className="field-error">{errors.confirmPassword.message}</span>}
        </label>
        <div className="settings-actions">
          <button className="button button-primary" type="submit" disabled={change.isPending}>{change.isPending ? "Changing…" : "Change password"}</button>
          {saved && <span className="form-notice" role="status">Password changed. You stay signed in here.</span>}
        </div>
      </form>
    </section>
  );
}

function DeleteAccount() {
  const remove = useDeleteAccount();
  // A full page load drops every cached query of this user (see useDeleteAccount).
  const onConfirm = () => remove.mutate(undefined, { onSuccess: () => window.location.replace("/") });
  return (
    <section className="panel settings-card danger-zone" aria-labelledby="delete-heading">
      <h2 id="delete-heading">Delete account</h2>
      <p>This permanently deletes your account with its shelves, reading progress and history, ratings, and reviews. Books stay in the library. It cannot be undone.</p>
      <ConfirmButton
        className="button button-danger"
        label="Delete my account"
        question="Delete your account and all of your data for good?"
        confirmLabel="Yes, delete everything"
        pending={remove.isPending}
        onConfirm={onConfirm}
      />
      {remove.isError && <p className="field-error" role="alert">{errorMessage(remove.error)}</p>}
    </section>
  );
}
