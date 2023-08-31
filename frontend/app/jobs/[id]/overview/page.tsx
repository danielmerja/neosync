'use client';
import PageHeader from '@/components/headers/PageHeader';
import { PageProps } from '@/components/types';
import { Skeleton } from '@/components/ui/skeleton';
import { useGetJob } from '@/libs/hooks/useGetJob';

import { ReactElement } from 'react';

export default function Page({ params }: PageProps): ReactElement {
  const id = params?.id ?? '';
  const { data, isLoading } = useGetJob(id);

  if (isLoading) {
    return <Skeleton />;
  }

  return (
    <div className="job-details-container">
      <PageHeader header="Overview" description="View job details" />
      <div className="flex flex-col my-4 gap-2">
        <div className="flex flex-row justify-between">
          <div className="flex flex-row items-center gap-2">
            <h3 className="text-xl">Job Name:</h3>
          </div>
        </div>
        <h3 className="text-2xl font-medium">{data?.job?.name}</h3>
      </div>
    </div>
  );
}