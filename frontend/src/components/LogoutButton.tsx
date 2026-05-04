export function LogoutButton() {
  const handleLogout = async () => {
    const res = await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
    })
    if (res.ok) {
      const { redirect } = await res.json() as { redirect: string }
      window.location.href = redirect
    } else {
      window.location.href = '/'
    }
  }

  return (
    <button onClick={handleLogout} type="button">
      Sign out
    </button>
  )
}
