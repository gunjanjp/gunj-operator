// Example React Component with i18n
// This shows best practices for using internationalization in Gunj Operator UI

import React, { useState } from 'react';
import { useTranslation, Trans } from 'react-i18next';
import {
  Card,
  CardContent,
  CardActions,
  Typography,
  Button,
  Chip,
  IconButton,
  Menu,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Box,
} from '@mui/material';
import {
  Edit as EditIcon,
  Delete as DeleteIcon,
  MoreVert as MoreVertIcon,
  Language as LanguageIcon,
} from '@mui/icons-material';
import { formatDistanceToNow } from 'date-fns';
import { enUS, ja, es } from 'date-fns/locale';

// Type definitions
interface Platform {
  name: string;
  namespace: string;
  status: 'ready' | 'installing' | 'failed' | 'upgrading' | 'unknown';
  componentCount: number;
  lastUpdated: Date;
}

interface PlatformCardProps {
  platform: Platform;
  onEdit: (platform: Platform) => void;
  onDelete: (platform: Platform) => void;
}

// Date locale mapping
const dateLocales = {
  en: enUS,
  ja: ja,
  es: es,
};

// Status color mapping
const statusColors = {
  ready: 'success',
  installing: 'info',
  failed: 'error',
  upgrading: 'warning',
  unknown: 'default',
} as const;

export const PlatformCard: React.FC<PlatformCardProps> = ({
  platform,
  onEdit,
  onDelete,
}) => {
  const { t, i18n } = useTranslation();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  // Format relative time with locale support
  const formatRelativeTime = (date: Date) => {
    const locale = dateLocales[i18n.language as keyof typeof dateLocales] || enUS;
    return formatDistanceToNow(date, { addSuffix: true, locale });
  };

  const handleMenuOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleEdit = () => {
    handleMenuClose();
    onEdit(platform);
  };

  const handleDeleteClick = () => {
    handleMenuClose();
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = () => {
    setDeleteDialogOpen(false);
    onDelete(platform);
  };

  const handleDeleteCancel = () => {
    setDeleteDialogOpen(false);
  };

  return (
    <>
      <Card>
        <CardContent>
          {/* Platform Title */}
          <Box display="flex" justifyContent="space-between" alignItems="flex-start">
            <Box>
              <Typography variant="h6" component="h2">
                {platform.name}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {platform.namespace}
              </Typography>
            </Box>
            
            {/* Status Chip with Translation */}
            <Chip
              label={t(`platform.status.${platform.status}`)}
              color={statusColors[platform.status]}
              size="small"
            />
          </Box>

          {/* Component Count with Pluralization */}
          <Typography variant="body2" sx={{ mt: 2 }}>
            {t('platform.component_count', { count: platform.componentCount })}
          </Typography>

          {/* Last Updated with Relative Time */}
          <Typography variant="caption" color="text.secondary">
            {formatRelativeTime(platform.lastUpdated)}
          </Typography>
        </CardContent>

        <CardActions>
          {/* Direct Action Buttons */}
          <Button
            size="small"
            startIcon={<EditIcon />}
            onClick={handleEdit}
          >
            {t('platform.edit')}
          </Button>

          {/* More Actions Menu */}
          <IconButton
            size="small"
            onClick={handleMenuOpen}
            aria-label={t('common.actions')}
          >
            <MoreVertIcon />
          </IconButton>
          <Menu
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={handleMenuClose}
          >
            <MenuItem onClick={handleDeleteClick}>
              <DeleteIcon sx={{ mr: 1 }} />
              {t('platform.delete')}
            </MenuItem>
          </Menu>
        </CardActions>
      </Card>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={handleDeleteCancel}
        aria-labelledby="delete-dialog-title"
        aria-describedby="delete-dialog-description"
      >
        <DialogTitle id="delete-dialog-title">
          {t('platform.delete')}
        </DialogTitle>
        <DialogContent>
          <DialogContentText id="delete-dialog-description">
            {/* Using Trans component for complex translations with variables */}
            <Trans
              i18nKey="platform.confirmDelete"
              values={{ name: platform.name }}
              components={{
                strong: <strong />,
              }}
            />
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDeleteCancel}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleDeleteConfirm} color="error" autoFocus>
            {t('common.yes')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

// Language Switcher Component
export const LanguageSwitcher: React.FC = () => {
  const { i18n } = useTranslation();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const languages = [
    { code: 'en', name: 'English', flag: '🇺🇸' },
    { code: 'ja', name: '日本語', flag: '🇯🇵' },
    { code: 'es', name: 'Español', flag: '🇪🇸' },
  ];

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleLanguageChange = (langCode: string) => {
    i18n.changeLanguage(langCode);
    handleClose();
    
    // Persist language preference
    localStorage.setItem('preferred-language', langCode);
  };

  const currentLanguage = languages.find(lang => lang.code === i18n.language) || languages[0];

  return (
    <>
      <Button
        startIcon={<LanguageIcon />}
        endIcon={<span>{currentLanguage.flag}</span>}
        onClick={handleClick}
        variant="outlined"
        size="small"
      >
        {currentLanguage.name}
      </Button>
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        {languages.map((lang) => (
          <MenuItem
            key={lang.code}
            onClick={() => handleLanguageChange(lang.code)}
            selected={lang.code === i18n.language}
          >
            <Box display="flex" alignItems="center" gap={1}>
              <span>{lang.flag}</span>
              <span>{lang.name}</span>
            </Box>
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};

// Example usage in a parent component
export const PlatformListExample: React.FC = () => {
  const { t } = useTranslation();
  
  // Example data
  const platforms: Platform[] = [
    {
      name: 'production',
      namespace: 'monitoring',
      status: 'ready',
      componentCount: 4,
      lastUpdated: new Date(Date.now() - 1000 * 60 * 30), // 30 minutes ago
    },
    {
      name: 'staging',
      namespace: 'monitoring',
      status: 'installing',
      componentCount: 3,
      lastUpdated: new Date(Date.now() - 1000 * 60 * 60 * 2), // 2 hours ago
    },
  ];

  const handleEdit = (platform: Platform) => {
    console.log('Edit platform:', platform);
  };

  const handleDelete = (platform: Platform) => {
    console.log('Delete platform:', platform);
  };

  return (
    <Box>
      {/* Page Header with Language Switcher */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4">
          {t('platform.title')}
        </Typography>
        <LanguageSwitcher />
      </Box>

      {/* Platform Cards */}
      <Box display="grid" gap={2} gridTemplateColumns="repeat(auto-fill, minmax(300px, 1fr))">
        {platforms.map((platform) => (
          <PlatformCard
            key={`${platform.namespace}/${platform.name}`}
            platform={platform}
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
        ))}
      </Box>
    </Box>
  );
};