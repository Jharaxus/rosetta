<#ftl output_format="HTML">
<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Connexion — Rosetta</title>
  <link rel="stylesheet" href="${url.resourcesPath}/css/login.css">
</head>
<body>
  <div class="kc-login-page">
    <div class="kc-inner">

      <div class="kc-logo">
        <div class="kc-logo-dot"></div>
        <span class="kc-logo-text">Rosetta</span>
      </div>

      <div class="kc-card">

        <#if message?has_content>
          <div class="kc-alert kc-alert-${message.type}">
            ${kcSanitize(message.summary)?no_esc}
          </div>
        </#if>

        <form class="kc-form" action="${url.loginAction?no_esc}" method="post">

          <#if !usernameHidden??>
          <div class="kc-field">
            <label class="kc-label" for="username">
              <#if !realm.loginWithEmailAllowed>Nom d'utilisateur
              <#elseif !realm.registrationEmailAsUsername>Email ou nom d'utilisateur
              <#else>Email</#if>
            </label>
            <input
              class="kc-input <#if messagesPerField.existsError('username','password')>kc-input-error</#if>"
              type="text"
              id="username"
              name="username"
              value="${login.username!''}"
              autofocus
              autocomplete="username"
            />
            <#if messagesPerField.existsError('username','password')>
              <span class="kc-field-error">${kcSanitize(messagesPerField.getFirstError('username','password'))?no_esc}</span>
            </#if>
          </div>
          </#if>

          <div class="kc-field">
            <label class="kc-label" for="password">Mot de passe</label>
            <input
              class="kc-input <#if messagesPerField.existsError('username','password')>kc-input-error</#if>"
              type="password"
              id="password"
              name="password"
              autocomplete="current-password"
            />
          </div>

          <#if realm.rememberMe && !usernameHidden??>
          <div class="kc-remember">
            <input
              type="checkbox"
              id="rememberMe"
              name="rememberMe"
              <#if login.rememberMe?? && login.rememberMe>checked</#if>
            />
            <label for="rememberMe">Se souvenir de moi</label>
          </div>
          </#if>

          <input type="hidden" id="id-hidden-input" name="credentialId"
                 <#if auth.selectedCredential?has_content>value="${auth.selectedCredential}"</#if> />

          <button class="kc-btn-primary" type="submit" name="login">
            Se connecter
            <span class="kc-btn-arrow">→</span>
          </button>

        </form>

        <div class="kc-footer">
          <#if realm.resetPasswordAllowed>
            <a class="kc-link kc-link-accent" href="${url.loginResetCredentialsUrl}">
              Mot de passe oublié ?
            </a>
          </#if>
        </div>

      </div>

    </div>
  </div>
</body>
</html>
