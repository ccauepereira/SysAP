"use client";

export function LogoutButton() {
  return (
    <button
      type="button"
      onClick={async () => {
        await fetch("/api/auth/logout", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
        });
        window.location.replace("/login");
      }}
    >
      Sair
    </button>
  );
}
