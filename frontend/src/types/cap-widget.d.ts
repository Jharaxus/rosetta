declare global {
  namespace JSX {
    interface IntrinsicElements {
      'cap-widget': React.ClassAttributes<HTMLElement> & React.HTMLAttributes<HTMLElement> & {
        'data-cap-api-endpoint'?: string
      }
    }
  }
}

export {}
