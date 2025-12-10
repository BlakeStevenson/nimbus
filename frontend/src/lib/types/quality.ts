// Quality Profile System Types

export interface QualityDefinition {
  id: number;
  name: string;
  title: string;
  resolution?: number;
  source?: string;
  modifier?: string;
  min_size?: number;
  max_size?: number;
  weight: number;
  created_at: string;
  updated_at: string;
}

export interface QualityProfileItem {
  id: number;
  profile_id: number;
  quality_id: number;
  allowed: boolean;
  sort_order: number;
  created_at: string;
  quality?: QualityDefinition;
}

export interface QualityProfile {
  id: number;
  name: string;
  description?: string;
  cutoff_quality_id?: number;
  upgrade_allowed: boolean;
  created_at: string;
  updated_at: string;
  items?: QualityProfileItem[];
  cutoff_quality?: QualityDefinition;
}

export interface MediaQuality {
  id: number;
  media_item_id: number;
  media_file_id?: number;
  quality_id?: number;
  profile_id?: number;
  detected_quality?: string;
  resolution?: number;
  source?: string;
  codec_video?: string;
  codec_audio?: string;
  is_proper: boolean;
  is_repack: boolean;
  is_remux: boolean;
  revision_version: number;
  upgrade_allowed: boolean;
  cutoff_met: boolean;
  created_at: string;
  updated_at: string;
  quality?: QualityDefinition;
  profile?: QualityProfile;
}

export interface QualityUpgradeHistory {
  id: number;
  media_item_id: number;
  old_quality_id?: number;
  new_quality_id?: number;
  old_file_id?: number;
  new_file_id?: number;
  download_id?: string;
  reason?: string;
  old_file_size?: number;
  new_file_size?: number;
  created_at: string;
  created_by_user_id?: number;
  old_quality?: QualityDefinition;
  new_quality?: QualityDefinition;
}

export interface DetectedQualityInfo {
  quality?: QualityDefinition;
  quality_name: string;
  resolution?: number;
  source?: string;
  codec_video?: string;
  codec_audio?: string;
  is_proper: boolean;
  is_repack: boolean;
  is_remux: boolean;
  is_remastered: boolean;
}

export interface CreateQualityDefinitionParams {
  name: string;
  title: string;
  resolution?: number;
  source?: string;
  modifier?: string;
  min_size?: number;
  max_size?: number;
  weight: number;
}

export interface UpdateQualityDefinitionParams {
  title?: string;
  resolution?: number;
  source?: string;
  modifier?: string;
  min_size?: number;
  max_size?: number;
  weight?: number;
}

export interface CreateQualityProfileItemParams {
  quality_id: number;
  allowed: boolean;
  sort_order: number;
}

export interface CreateQualityProfileParams {
  name: string;
  description?: string;
  cutoff_quality_id?: number;
  upgrade_allowed: boolean;
  items?: CreateQualityProfileItemParams[];
}

export interface UpdateQualityProfileParams {
  name?: string;
  description?: string;
  cutoff_quality_id?: number;
  upgrade_allowed?: boolean;
  items?: CreateQualityProfileItemParams[];
}

export interface AssignProfileToMediaParams {
  profile_id: number;
}

export interface QualityUpgradeCheckResult {
  can_upgrade: boolean;
  current_quality?: QualityDefinition;
  available_quality?: QualityDefinition;
  reason?: string;
}

export interface MediaForUpgradeResponse {
  media_ids: number[];
  count: number;
}
