/**
 * Realistic sample emails for extraction testing.
 *
 * `code`/`link` are the expected extraction results (null = must extract nothing).
 * Includes deliberately tricky cases: order numbers, phone numbers, dates,
 * prices, years, and decoy links (unsubscribe, social, tracking).
 */
export interface SampleEmail {
  name: string;
  subject: string;
  text: string;
  html?: string;
  code: string | null;
  link: string | null;
}

export const SAMPLE_EMAILS: SampleEmail[] = [
  {
    name: "github-verify-code",
    subject: "[GitHub] Please verify your device",
    text: "Hey! A sign in attempt requires further verification. Your verification code is 628139. If you did not attempt to sign in, change your password.",
    code: "628139",
    link: null,
  },
  {
    name: "google-g-code",
    subject: "Your Google verification code",
    text: "Your Google verification code is G-472918. Do not share this code with anyone.",
    code: "472918",
    link: null,
  },
  {
    name: "netflix-code",
    subject: "Your Netflix temporary access code",
    text: "Enter this code to finish signing in: 4821. This code expires in 15 minutes.",
    code: "4821",
    link: null,
  },
  {
    name: "slack-grouped-code",
    subject: "Slack confirmation code: 824-193",
    text: "Your Slack confirmation code is 824-193. Enter it in the window where you started signing in.",
    code: "824193",
    link: null,
  },
  {
    name: "shopify-code",
    subject: "Your login code",
    text: "Use the following code to log in: 913204. The code is valid for 10 minutes.",
    code: "913204",
    link: null,
  },
  {
    name: "stripe-otp",
    subject: "Your Stripe one-time passcode",
    text: "Your one-time passcode is 55018. It expires shortly.",
    code: "55018",
    link: null,
  },
  {
    name: "discord-code",
    subject: "Discord",
    text: "Your Discord verification code is: 390145",
    code: "390145",
    link: null,
  },
  {
    name: "bank-otp-plain",
    subject: "Your security code",
    text: "Use 837261 to complete your sign-in. Never share this one-time passcode.",
    code: "837261",
    link: null,
  },
  {
    name: "amazon-otp",
    subject: "Amazon",
    text: "To verify your email address, please use the following One Time Password (OTP): 429105",
    code: "429105",
    link: null,
  },
  {
    name: "apple-id-code",
    subject: "Your Apple ID Verification Code",
    text: "Your Apple ID Code is: 512904. Do not share it with anyone.",
    code: "512904",
    link: null,
  },
  {
    name: "microsoft-code",
    subject: "Microsoft account security code",
    text: "Security code: 7391042\n\nIf you don't recognize this activity, someone else may have your password.",
    code: "7391042",
    link: null,
  },
  {
    name: "twitch-code",
    subject: "Twitch verification",
    text: "Here is your one-time code: 204815. It expires in 10 minutes.",
    code: "204815",
    link: null,
  },
  {
    name: "paypal-code",
    subject: "Here is your PayPal security code",
    text: "Your security code is 490217. It will expire in 5 minutes.",
    code: "490217",
    link: null,
  },
  {
    name: "code-in-subject-only",
    subject: "482913 is your verification code",
    text: "Enter it to continue.",
    code: "482913",
    link: null,
  },
  {
    name: "alphanumeric-code",
    subject: "Your access code",
    text: "Your one-time code is A1B2C3. Enter it to continue.",
    code: "A1B2C3",
    link: null,
  },
  {
    name: "grouped-4-4-code",
    subject: "Confirm your identity",
    text: "Your verification code is 4821 9930.",
    code: "48219930",
    link: null,
  },

  // --- links ----------------------------------------------------------------
  {
    name: "github-verify-link",
    subject: "Confirm your email address",
    text: "Almost done! Confirm your email address by clicking the link below:\nhttps://github.com/users/confirm?token=abc123def456",
    code: null,
    link: "https://github.com/users/confirm?token=abc123def456",
  },
  {
    name: "magic-login-link",
    subject: "Your magic sign-in link",
    text: "Click to sign in: https://app.example.com/magic/login?token=9f8e7d6c",
    code: null,
    link: "https://app.example.com/magic/login?token=9f8e7d6c",
  },
  {
    name: "password-reset-link",
    subject: "Reset your password",
    text: "We received a request to reset your password. Reset it here: https://accounts.example.com/reset?token=xyz987\nIf you didn't ask for this, ignore this email.",
    code: null,
    link: "https://accounts.example.com/reset?token=xyz987",
  },
  {
    name: "activate-account-html",
    subject: "Activate your account",
    text: "",
    html: '<p>Welcome aboard!</p><a href="https://example.com/activate?id=7721&k=aabbcc">Activate your account</a><br><a href="https://example.com/unsubscribe?u=55">Unsubscribe</a>',
    code: null,
    link: "https://example.com/activate?id=7721&k=aabbcc",
  },
  {
    name: "anchor-text-is-the-signal",
    subject: "Finish signing up",
    text: "",
    html: '<a href="https://t.example.com/c/9z8y7x">Verify your email</a> <a href="https://example.com/privacy">Privacy Policy</a>',
    code: null,
    link: "https://t.example.com/c/9z8y7x",
  },
  {
    name: "html-entities-in-url",
    subject: "Confirm subscription",
    text: "",
    html: '<a href="https://example.com/confirm?a=1&amp;b=2&amp;token=qq">Confirm your subscription</a>',
    code: null,
    link: "https://example.com/confirm?a=1&b=2&token=qq",
  },
  {
    name: "invite-link",
    subject: "You've been invited",
    text: "Jane invited you to the team. Accept your invitation: https://example.com/invite/accept?code=team42",
    code: null,
    link: "https://example.com/invite/accept?code=team42",
  },

  // --- tricky: must NOT extract a wrong code/link ---------------------------
  {
    name: "order-number-decoy",
    subject: "Your order #90210 has shipped",
    text: "Thanks for your order! Order number 90210 shipped on 03/14/2025. Track it anytime.",
    code: null,
    link: null,
  },
  {
    name: "order-plus-real-code",
    subject: "Order confirmation",
    text: "Order #90210 confirmed. Your verification code is 55512 to approve the change.",
    code: "55512",
    link: null,
  },
  {
    name: "phone-number-decoy",
    subject: "We're here to help",
    text: "Questions? Call us at 1-800-555-1234 any time, or reply to this email.",
    code: null,
    link: null,
  },
  {
    name: "price-decoy",
    subject: "Your receipt",
    text: "Your total is $1,234.56 charged on 2025-10-21. Thanks for your purchase!",
    code: null,
    link: null,
  },
  {
    name: "year-decoy",
    subject: "Our 2025 year in review",
    text: "Copyright 2024 Example Inc. Here's everything that happened in 2025.",
    code: null,
    link: null,
  },
  {
    name: "date-decoy",
    subject: "Appointment reminder",
    text: "Your appointment is scheduled for 2025-11-30 at 14:30. See you then!",
    code: null,
    link: null,
  },
  {
    name: "newsletter-no-code-no-link",
    subject: "This week in tech",
    text: "Thanks for reading our weekly newsletter. Catch up on the latest stories and trends.",
    code: null,
    link: null,
  },
  {
    name: "unsubscribe-only-link",
    subject: "Weekly digest",
    text: "Here are your updates for the week. To stop receiving these, unsubscribe: https://example.com/unsubscribe?u=99",
    code: null,
    link: null,
  },
  {
    name: "social-links-only",
    subject: "Follow us",
    text: "Stay connected! https://twitter.com/example https://facebook.com/example https://instagram.com/example",
    code: null,
    link: null,
  },
  {
    name: "tracking-pixel-decoy-html",
    subject: "Promo inside",
    text: "",
    html: '<img src="https://track.example.com/open.gif?id=1"/><a href="https://example.com/sale">Shop the sale</a>',
    code: null,
    link: null,
  },
];
