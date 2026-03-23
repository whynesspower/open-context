import type { HTMLAttributes, ReactNode } from 'react';

import { cn } from '@/lib/cn';

type CardProps = HTMLAttributes<HTMLDivElement> & {
  title?: string;
  description?: string;
  icon?: ReactNode;
};

export function Card({ title, description, icon, className, children, ...props }: CardProps) {
  return (
    <section className={cn('card', className)} {...props}>
      {title || description || icon ? (
        <div className="card-header">
          <div>
            {title ? <h2 className="card-title">{title}</h2> : null}
            {description ? <p className="card-description">{description}</p> : null}
          </div>
          {icon ? <div className="card-icon">{icon}</div> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}
